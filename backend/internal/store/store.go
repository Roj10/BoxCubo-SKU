// ============================================================
//  STORE: a "despensa" — onde os dados são guardados e lidos
// ============================================================
//  Tudo que o sistema sabe (máquinas, peças, listas de materiais,
//  movimentações) fica num único arquivo: backend/data/boxcubo.json.
//  Este arquivo é o único que mexe nele. Ninguém mais lê ou grava o
//  arquivo direto — todo mundo pede para o "store".
//
//  Por que isso é bom? Quando a BoxCubo escolher um banco de dados de
//  verdade (PostgreSQL, MySQL...), só este arquivo precisa ser refeito.
//  A tela e as regras continuam iguais.
//
//  Duas ideias importantes aqui:
//   - TRAVA (mu): se duas pessoas salvarem ao mesmo tempo, uma espera a
//     outra terminar, para o arquivo nunca ficar pela metade.
//   - GRAVAÇÃO SEGURA (save): grava primeiro num arquivo temporário e só
//     depois troca pelo verdadeiro. Se faltar luz no meio, o arquivo antigo
//     continua inteiro.

// Package store guarda os dados do sistema.
//
// Store é a interface usada pela API. Hoje a única implementação é FileStore,
// que grava tudo num arquivo JSON local (data/boxcubo.json). Quando o banco
// for escolhido (PostgreSQL, MySQL, SQLite...), basta criar outra
// implementação desta interface e trocar no main.go — a API não muda.
package store

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"boxcubo-sku/internal/model"
)

var (
	ErrNotFound = errors.New("não encontrado")
	ErrConflict = errors.New("conflito")
	ErrInvalid  = errors.New("dados inválidos")
)

// Store é o contrato de persistência.
// A "lista de serviços" que a despensa oferece. É um contrato: qualquer
// outro tipo de armazenamento (um banco de dados no futuro) precisa saber
// fazer exatamente estas coisas.
type Store interface {
	ListMachines() []model.Machine
	GetMachine(id string) (model.Machine, error)
	SetMinimo(machineID string, maquinas int, comKits bool) (model.Machine, error)
	ListKits() []model.Kit
	Legenda() map[string]string

	ListParts(q string) []model.Part
	GetPart(sku string) (model.Part, error)
	// CreatePart cria a peça. sufixo é o final do SKU (ex.: "1K2"); vazio = número automático.
	CreatePart(prefixo, sufixo string, p model.Part) (model.Part, error)
	// RenameSKU troca o código da peça em todo lugar (BOM, movimentações, foto).
	RenameSKU(antigo, novo string) (model.Part, error)
	UpdatePart(sku string, p model.Part) (model.Part, error)
	DeletePart(sku string) error
	ListClasses() []model.Classe
	ListFornecedores() []string
	ListSubconjuntos() []model.Part
	// RenomearFornecedor troca o nome em todas as peças; se o nome novo já
	// existir numa peça, junta os dois (serve para unir duplicados).
	RenomearFornecedor(de, para string) (int, error)
	// SubstituirFornecedor serve para quando a empresa não trabalha mais com
	// um fornecedor: passa as peças dele para outro (sem o código antigo, que
	// era do fornecedor velho) ou, com "para" vazio, só tira ele das peças.
	// Devolve quantas peças mudaram e quais ficaram sem nenhum fornecedor.
	SubstituirFornecedor(de, para string) (int, []string, error)
	// SetCodigoFornecedor grava o código do item da peça naquele fornecedor.
	SetCodigoFornecedor(sku, fornecedor, codigo string) (model.Part, error)
	SetPartImage(sku, url string) (model.Part, error)

	GetBOM(machineID string) ([]model.BOMItem, error)
	UpsertBOMItem(machineID string, item model.BOMItem) error
	DeleteBOMItem(machineID, sku string) error

	// Movimentações de estoque.
	ListMovements() []model.Movement
	GetMovement(id string) (model.Movement, error)
	// RegisterMovement aplica as mudanças de estoque e grava o histórico numa
	// única operação. Saídas maiores que o estoque tiram o que existe, deixam a
	// peça em 0 e registram a falta.
	RegisterMovement(m model.Movement, changes []model.StockChange) (model.Movement, error)
	// ReverseMovement desfaz uma movimentação (devolve/retira o que ela mexeu).
	ReverseMovement(id, observacao string) (model.Movement, error)
	// SerialEmUso diz se o número de série já saiu numa movimentação ativa.
	SerialEmUso(serial string) (movID string, emUso bool)
}

//go:embed seed.json
var seedJSON []byte

// Tudo que fica dentro do arquivo boxcubo.json, organizado por assunto.
type data struct {
	Machines  []model.Machine            `json:"machines"`
	Kits      []model.Kit                `json:"kits"`
	Parts     []model.Part               `json:"parts"`
	BOM       map[string][]model.BOMItem `json:"bom"`
	Legenda   map[string]string          `json:"legenda"`
	Movements []model.Movement           `json:"movements"`
	Migracoes []string                   `json:"migracoes"` // migrações de dados já aplicadas
}

// FileStore guarda os dados num arquivo JSON local.
// FileStore é a despensa que guarda tudo num arquivo JSON local.
type FileStore struct {
	mu   sync.RWMutex
	path string
	d    data
}

// NewFileStore abre o arquivo; se ele não existir, cria a partir do seed da V2.
// Abre o arquivo de dados. Se ele não existir, começa com o "seed" (as 123
// peças da V2 que vieram da planilha) e já grava o arquivo.
func NewFileStore(path string) (*FileStore, error) {
	s := &FileStore{path: path}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw = seedJSON
	} else if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.d); err != nil {
		return nil, fmt.Errorf("lendo %s: %w", path, err)
	}
	if s.d.BOM == nil {
		s.d.BOM = map[string][]model.BOMItem{}
	}
	if err := s.save(); err != nil {
		return nil, err
	}
	return s, nil
}

// save grava de forma atômica (arquivo temporário + rename).
// Grava tudo no disco de forma segura (temporário + troca).
func (s *FileStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.d, "", " ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// ---------------------------- MÁQUINAS ----------------------------
// Lista as máquinas da barra lateral.
func (s *FileStore) ListMachines() []model.Machine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]model.Machine(nil), s.d.Machines...)
}

func (s *FileStore) GetMachine(id string) (model.Machine, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.d.Machines {
		if m.ID == id {
			return m, nil
		}
	}
	return model.Machine{}, ErrNotFound
}

// Salva o estoque mínimo da máquina ("peças para N máquinas").
func (s *FileStore) SetMinimo(machineID string, maquinas int, comKits bool) (model.Machine, error) {
	if maquinas < 0 {
		return model.Machine{}, fmt.Errorf("%w: mínimo negativo", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.d.Machines {
		if s.d.Machines[i].ID == machineID {
			s.d.Machines[i].MinimoMaquinas = maquinas
			s.d.Machines[i].MinimoComKits = comKits
			return s.d.Machines[i], s.save()
		}
	}
	return model.Machine{}, ErrNotFound
}

func (s *FileStore) ListKits() []model.Kit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]model.Kit(nil), s.d.Kits...)
}

func (s *FileStore) Legenda() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.d.Legenda))
	for k, v := range s.d.Legenda {
		out[k] = v
	}
	return out
}

// ----------------------------- PEÇAS ------------------------------
// Lista as peças. Se vier um texto de busca, devolve só as que têm esse
// texto no SKU, nome, classe, fornecedor ou código do fornecedor.
func (s *FileStore) ListParts(q string) []model.Part {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = strings.ToLower(strings.TrimSpace(q))
	out := []model.Part{}
	for _, p := range s.d.Parts {
		txt := p.SKU + " " + p.Nome + " " + p.Classe
		for _, f := range p.Fornecedores { // dá para buscar pelo código do fornecedor
			txt += " " + f.Nome + " " + f.Codigo
		}
		if q == "" || strings.Contains(strings.ToLower(txt), q) {
			out = append(out, p)
		}
	}
	return out
}

// Procura a posição da peça pelo SKU (-1 = não achou).
func (s *FileStore) partIndex(sku string) int {
	for i, p := range s.d.Parts {
		if p.SKU == sku {
			return i
		}
	}
	return -1
}

func (s *FileStore) GetPart(sku string) (model.Part, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i := s.partIndex(sku); i >= 0 {
		return s.d.Parts[i], nil
	}
	return model.Part{}, ErrNotFound
}

// ListFornecedores devolve os nomes de fornecedores já usados (para sugerir na tela).
// Lista os nomes de fornecedores já usados, para sugerir na tela
// (evita cadastrar "Fornell" e "FORNELL" como se fossem dois).
func (s *FileStore) ListFornecedores() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vistos := map[string]bool{}
	out := []string{}
	for _, p := range s.d.Parts {
		for _, f := range p.Fornecedores {
			if f.Nome != "" && !vistos[f.Nome] {
				vistos[f.Nome] = true
				out = append(out, f.Nome)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (s *FileStore) RenomearFornecedor(de, para string) (int, error) {
	de, para = strings.TrimSpace(de), strings.TrimSpace(para)
	if de == "" || para == "" {
		return 0, fmt.Errorf("%w: informe o nome atual e o novo", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for i := range s.d.Parts {
		p := &s.d.Parts[i]
		idxDe, idxPara := -1, -1
		for j, f := range p.Fornecedores {
			if f.Nome == de {
				idxDe = j
			}
			if f.Nome == para {
				idxPara = j
			}
		}
		if idxDe < 0 {
			continue
		}
		n++
		if idxPara < 0 || idxPara == idxDe {
			p.Fornecedores[idxDe].Nome = para
			continue
		}
		// a peça já tinha os dois: fica um só (na posição do primeiro),
		// guardando o código que estiver preenchido
		manter, tirar := idxPara, idxDe
		if idxDe < idxPara {
			manter, tirar = idxDe, idxPara
		}
		codigo := p.Fornecedores[idxPara].Codigo
		if codigo == "" {
			codigo = p.Fornecedores[idxDe].Codigo
		}
		p.Fornecedores[manter] = model.Fornecimento{Nome: para, Codigo: codigo}
		p.Fornecedores = append(p.Fornecedores[:tirar], p.Fornecedores[tirar+1:]...)
	}
	if n == 0 {
		return 0, ErrNotFound
	}
	return n, s.save()
}

func (s *FileStore) SubstituirFornecedor(de, para string) (int, []string, error) {
	de, para = strings.TrimSpace(de), strings.TrimSpace(para)
	if de == "" || de == para {
		return 0, nil, fmt.Errorf("%w: escolha um fornecedor diferente", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n, semFornecedor := 0, []string{}
	for i := range s.d.Parts {
		p := &s.d.Parts[i]
		idxDe, idxPara := -1, -1
		for j, f := range p.Fornecedores {
			if f.Nome == de {
				idxDe = j
			}
			if para != "" && f.Nome == para {
				idxPara = j
			}
		}
		if idxDe < 0 {
			continue
		}
		n++
		switch {
		case para == "":
			// só tira o fornecedor antigo
			p.Fornecedores = append(p.Fornecedores[:idxDe], p.Fornecedores[idxDe+1:]...)
		case idxPara >= 0:
			// a peça já tinha o novo: ele assume o lugar do antigo (se o
			// antigo era o principal, o novo vira principal), com o código dele
			novo := p.Fornecedores[idxPara]
			p.Fornecedores[idxDe] = novo
			p.Fornecedores = append(p.Fornecedores[:idxPara], p.Fornecedores[idxPara+1:]...)
		default:
			// troca pelo novo, sem código (o código era do fornecedor antigo)
			p.Fornecedores[idxDe] = model.Fornecimento{Nome: para}
		}
		if len(p.Fornecedores) == 0 {
			semFornecedor = append(semFornecedor, p.SKU)
		}
	}
	if n == 0 {
		return 0, nil, ErrNotFound
	}
	return n, semFornecedor, s.save()
}

func (s *FileStore) SetCodigoFornecedor(sku, fornecedor, codigo string) (model.Part, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.partIndex(sku)
	if i < 0 {
		return model.Part{}, ErrNotFound
	}
	for j, f := range s.d.Parts[i].Fornecedores {
		if f.Nome == fornecedor {
			s.d.Parts[i].Fornecedores[j].Codigo = strings.TrimSpace(codigo)
			return s.d.Parts[i], s.save()
		}
	}
	return model.Part{}, fmt.Errorf("%w: %s não é fornecedor de %s", ErrNotFound, fornecedor, sku)
}

// ListClasses devolve os prefixos de SKU existentes (BC-ELE-RES, ...).
// Lista os tipos de peça existentes (ELE-RES, EST-PAR...) para o
// formulário de peça nova.
func (s *FileStore) ListClasses() []model.Classe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]model.Classe{}
	for _, p := range s.d.Parts {
		pre := prefixOf(p.SKU)
		if _, ok := seen[pre]; !ok {
			seen[pre] = model.Classe{Prefixo: pre, Grupo: p.Grupo, Classe: p.Classe}
		}
	}
	out := make([]model.Classe, 0, len(seen))
	for _, c := range seen {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Prefixo < out[j].Prefixo })
	return out
}

// Corta o final do SKU: ELE-RES-10 -> ELE-RES.
func prefixOf(sku string) string {
	if i := strings.LastIndex(sku, "-"); i > 0 {
		return sku[:i]
	}
	return sku
}

var sufixoValido = regexp.MustCompile(`^[A-Z0-9]{1,16}$`)

// NormalizarSufixo deixa o final do SKU em maiúsculas e valida (só letras e números).
// Arruma e confere o "final" do SKU: deixa em maiúsculas e só aceita
// letras e números (o hífen é reservado para separar as partes).
func NormalizarSufixo(sufixo string) (string, error) {
	sufixo = strings.ToUpper(strings.TrimSpace(sufixo))
	if !sufixoValido.MatchString(sufixo) {
		return "", fmt.Errorf("%w: o final do SKU deve ter de 1 a 16 letras ou números, sem espaços nem hífen (ex.: 1K2, M4X10)", ErrInvalid)
	}
	return sufixo, nil
}

// CreatePart cria a peça com o final escolhido (BC-ELE-RES-47) ou, se vazio,
// com o próximo número livre do prefixo (BC-ELE-RES-013).
// Cadastra uma peça nova. Se a pessoa escolheu o final (ex. "47"), o SKU
// fica ELE-RES-47; se não, o sistema pega o primeiro número livre (-001...).
func (s *FileStore) CreatePart(prefixo, sufixo string, p model.Part) (model.Part, error) {
	prefixo = strings.ToUpper(strings.TrimSpace(prefixo))
	if prefixo == "" || strings.TrimSpace(p.Nome) == "" {
		return model.Part{}, fmt.Errorf("%w: prefixo e nome são obrigatórios", ErrInvalid)
	}
	if strings.TrimSpace(sufixo) != "" {
		var err error
		if sufixo, err = NormalizarSufixo(sufixo); err != nil {
			return model.Part{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if sufixo != "" {
		p.SKU = prefixo + "-" + sufixo
		if s.partIndex(p.SKU) >= 0 {
			return model.Part{}, fmt.Errorf("%w: o SKU %s já existe", ErrConflict, p.SKU)
		}
	} else {
		// primeiro número livre a partir de 001 (alguns finais descritivos
		// também são números, como o resistor -100, então não dá para usar "maior + 1")
		for n := 1; ; n++ {
			p.SKU = fmt.Sprintf("%s-%03d", prefixo, n)
			if s.partIndex(p.SKU) < 0 {
				break
			}
		}
	}
	normalize(&p)
	s.d.Parts = append(s.d.Parts, p)
	return p, s.save()
}

// Troca o código (SKU) de uma peça EM TODO LUGAR: na peça, nas listas de
// materiais, no histórico de movimentações e no endereço da foto.
func (s *FileStore) RenameSKU(antigo, novo string) (model.Part, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.renameSKU(antigo, novo)
	if err != nil {
		return model.Part{}, err
	}
	return p, s.save()
}

// renameSKU troca o SKU em todas as referências (chamar com o lock).
func (s *FileStore) renameSKU(antigo, novo string) (model.Part, error) {
	i := s.partIndex(antigo)
	if i < 0 {
		return model.Part{}, ErrNotFound
	}
	if antigo == novo {
		return s.d.Parts[i], nil
	}
	if s.partIndex(novo) >= 0 {
		return model.Part{}, fmt.Errorf("%w: o SKU %s já existe", ErrConflict, novo)
	}
	p := &s.d.Parts[i]
	p.SKU = novo
	// a foto é salva com o nome do SKU: /api/imagens/BC-ELE-RES-001.png?v=...
	p.Imagem = strings.Replace(p.Imagem, "/api/imagens/"+antigo+".", "/api/imagens/"+novo+".", 1)
	if lista, ok := s.d.BOM[antigo]; ok { // subconjunto: a receita acompanha o novo código
		s.d.BOM[novo] = lista
		delete(s.d.BOM, antigo)
	}
	for mid := range s.d.BOM {
		for j := range s.d.BOM[mid] {
			if s.d.BOM[mid][j].SKU == antigo {
				s.d.BOM[mid][j].SKU = novo
			}
		}
	}
	for m := range s.d.Movements {
		for j := range s.d.Movements[m].Itens {
			if s.d.Movements[m].Itens[j].SKU == antigo {
				s.d.Movements[m].Itens[j].SKU = novo
			}
		}
	}
	return *p, nil
}

// RenomearArquivosImagem acompanha a troca de SKU na pasta de fotos.
// A foto é salva com o nome do SKU (ex. ELE-RES-10.png); quando o SKU
// muda, o arquivo da foto é renomeado junto.
func RenomearArquivosImagem(imgDir, antigo, novo string) {
	for _, ext := range []string{".jpg", ".png", ".gif", ".webp"} {
		de := filepath.Join(imgDir, antigo+ext)
		if _, err := os.Stat(de); err == nil {
			_ = os.Rename(de, filepath.Join(imgDir, novo+ext))
		}
	}
}

// Salva as alterações de uma peça (nome, preço, estoque, fornecedores...).
// O SKU e a foto não mudam por aqui: eles têm caminhos próprios.
func (s *FileStore) UpdatePart(sku string, p model.Part) (model.Part, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.partIndex(sku)
	if i < 0 {
		return model.Part{}, ErrNotFound
	}
	if strings.TrimSpace(p.Nome) == "" {
		return model.Part{}, fmt.Errorf("%w: nome é obrigatório", ErrInvalid)
	}
	p.SKU = sku                    // o SKU nunca muda
	p.Imagem = s.d.Parts[i].Imagem // a foto só muda pela rota de imagem
	normalize(&p)
	s.d.Parts[i] = p
	return p, s.save()
}

// SetPartImage grava a URL da foto da peça ("" remove).
// Guarda (ou apaga, se vier vazio) o endereço da foto da peça.
func (s *FileStore) SetPartImage(sku, url string) (model.Part, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.partIndex(sku)
	if i < 0 {
		return model.Part{}, ErrNotFound
	}
	s.d.Parts[i].Imagem = url
	return s.d.Parts[i], s.save()
}

// Apaga uma peça do cadastro. Proteção: se ela ainda está na lista de
// alguma máquina, NÃO deixa apagar (primeiro tire da máquina).
func (s *FileStore) DeletePart(sku string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.partIndex(sku)
	if i < 0 {
		return ErrNotFound
	}
	for mid, items := range s.d.BOM {
		for _, it := range items {
			if it.SKU == sku {
				return fmt.Errorf("%w: a peça está na lista de %s", ErrConflict, mid)
			}
		}
	}
	delete(s.d.BOM, sku) // se era um subconjunto, a receita dele vai junto
	s.d.Parts = append(s.d.Parts[:i], s.d.Parts[i+1:]...)
	return s.save()
}

// ----------------------- LISTA DE MATERIAIS -----------------------
// Devolve a lista de materiais (BOM) de uma máquina.
func (s *FileStore) GetBOM(machineID string) ([]model.BOMItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasMachine(machineID) {
		return nil, ErrNotFound
	}
	return append([]model.BOMItem(nil), s.d.BOM[machineID]...), nil
}

// hasMachine diz se "id" pode ter lista de materiais: uma máquina (v2, eye...)
// ou um subconjunto fabricado (pelo SKU, ex.: ELE-SUB-CPUV2).
func (s *FileStore) hasMachine(id string) bool {
	for _, m := range s.d.Machines {
		if m.ID == id {
			return true
		}
	}
	if i := s.partIndex(id); i >= 0 && s.d.Parts[i].Fabricado {
		return true
	}
	return false
}

// contem diz se o subconjunto "dono" usa "sku" em algum nível da receita
// (serve para impedir receita circular: CPU dentro do Shield dentro da CPU...).
func (s *FileStore) contem(dono, sku string, visitados map[string]bool) bool {
	if dono == sku {
		return true
	}
	if visitados[dono] {
		return false
	}
	visitados[dono] = true
	for _, it := range s.d.BOM[dono] {
		if s.contem(it.SKU, sku, visitados) {
			return true
		}
	}
	return false
}

// ListSubconjuntos devolve as peças fabricadas (CPU, Shield Raspberry...).
func (s *FileStore) ListSubconjuntos() []model.Part {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []model.Part{}
	for _, p := range s.d.Parts {
		if p.Fabricado {
			out = append(out, p)
		}
	}
	return out
}

// UpsertBOMItem adiciona a peça à máquina ou atualiza quantidade/uso/kit.
// Coloca uma peça na lista da máquina, ou atualiza se ela já estiver lá
// (quantidade por máquina, onde é usada, se é de kit).
func (s *FileStore) UpsertBOMItem(machineID string, item model.BOMItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasMachine(machineID) || s.partIndex(item.SKU) < 0 {
		return ErrNotFound
	}
	if s.contem(item.SKU, machineID, map[string]bool{}) {
		return fmt.Errorf("%w: %s não pode entrar na receita de %s (ficaria circular)", ErrInvalid, item.SKU, machineID)
	}
	if item.Quantidade < 0 {
		return fmt.Errorf("%w: quantidade negativa", ErrInvalid)
	}
	if item.OndeUsado == nil {
		item.OndeUsado = []string{}
	}
	items := s.d.BOM[machineID]
	for i := range items {
		if items[i].SKU == item.SKU {
			items[i] = item
			return s.save()
		}
	}
	s.d.BOM[machineID] = append(items, item)
	return s.save()
}

// Tira uma peça da lista da máquina (a peça continua cadastrada).
func (s *FileStore) DeleteBOMItem(machineID, sku string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.d.BOM[machineID]
	for i := range items {
		if items[i].SKU == sku {
			s.d.BOM[machineID] = append(items[:i], items[i+1:]...)
			return s.save()
		}
	}
	return ErrNotFound
}

// ---- movimentações ----

// ------------------------- MOVIMENTAÇÕES --------------------------
// Lista o histórico, do mais novo para o mais antigo.
func (s *FileStore) ListMovements() []model.Movement {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Movement, len(s.d.Movements))
	// mais recentes primeiro
	for i, m := range s.d.Movements {
		out[len(out)-1-i] = m
	}
	return out
}

func (s *FileStore) movIndex(id string) int {
	for i, m := range s.d.Movements {
		if m.ID == id {
			return i
		}
	}
	return -1
}

func (s *FileStore) GetMovement(id string) (model.Movement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i := s.movIndex(id); i >= 0 {
		return s.d.Movements[i], nil
	}
	return model.Movement{}, ErrNotFound
}

func (s *FileStore) SerialEmUso(serial string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serialEmUso(serial)
}

// Diz se um número de série já saiu numa saída de máquina que vale
// (saídas estornadas não contam, então o número fica livre de novo).
func (s *FileStore) serialEmUso(serial string) (string, bool) {
	serial = strings.ToUpper(strings.TrimSpace(serial))
	for _, m := range s.d.Movements {
		if m.Tipo != model.MovSaidaMaquina || m.Estornado {
			continue
		}
		for _, ns := range m.NumerosSerie {
			if strings.ToUpper(ns) == serial {
				return m.ID, true
			}
		}
	}
	return "", false
}

// Registra uma movimentação (entrada, saída...) e mexe no estoque, tudo de
// uma vez. Antes confere se o número de série já foi usado.
func (s *FileStore) RegisterMovement(m model.Movement, changes []model.StockChange) (model.Movement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m.Tipo == model.MovSaidaMaquina {
		for _, ns := range m.NumerosSerie {
			if id, ok := s.serialEmUso(ns); ok {
				return model.Movement{}, fmt.Errorf("%w: número de série %s já saiu em %s", ErrConflict, ns, id)
			}
		}
	}
	m, err := s.apply(m, changes)
	if err != nil {
		return model.Movement{}, err
	}
	return m, s.save()
}

// apply altera o estoque e acrescenta a movimentação (chamar com o lock).
// O "coração" do estoque. Para cada peça:
//   - soma (entrada) ou subtrai (saída) a quantidade;
//   - se for sair MAIS do que existe, tira só o que tem, deixa a peça em 0
//     (nunca negativo) e anota quanto faltou;
//   - anota estoque antes e depois, para o histórico.
//
// Depois dá um número à movimentação (MOV-000001, MOV-000002...).
func (s *FileStore) apply(m model.Movement, changes []model.StockChange) (model.Movement, error) {
	// valida tudo antes de mexer em qualquer peça
	for _, c := range changes {
		if s.partIndex(c.SKU) < 0 {
			return model.Movement{}, fmt.Errorf("%w: peça %s", ErrNotFound, c.SKU)
		}
	}
	m.ID = fmt.Sprintf("MOV-%06d", len(s.d.Movements)+1)
	m.Data = time.Now().Format(time.RFC3339)
	m.Itens = []model.MovementItem{}
	for _, c := range changes {
		if c.Delta == 0 {
			continue
		}
		p := &s.d.Parts[s.partIndex(c.SKU)]
		antes := p.Estoque
		mov := c.Delta
		falta := 0.0
		// Ex.: tem 27 balanças e pediram 30 -> saem 27, fica 0, falta 3.
		if antes+mov < 0 { // não há o suficiente: tira o que tem e fica em 0
			falta = round3(-(antes + mov))
			mov = -math.Max(antes, 0)
		}
		p.Estoque = round3(antes + mov)
		if falta > 0 {
			m.ComFalta = true
		}
		m.Itens = append(m.Itens, model.MovementItem{
			SKU: p.SKU, Nome: p.Nome, Unidade: p.Unidade,
			Solicitado: round3(math.Abs(c.Delta)), Movimentado: round3(mov),
			EstoqueAntes: antes, EstoqueDepois: p.Estoque, Falta: falta,
		})
	}
	s.d.Movements = append(s.d.Movements, m)
	return m, nil
}

// ESTORNO: desfaz uma movimentação lançada errada. Faz o caminho inverso
// exatamente do que ela mexeu (se saíram 27, voltam 27) e marca a original
// como "estornada". O histórico nunca é apagado, só corrigido.
func (s *FileStore) ReverseMovement(id, observacao string) (model.Movement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.movIndex(id)
	if i < 0 {
		return model.Movement{}, ErrNotFound
	}
	orig := s.d.Movements[i]
	if orig.Tipo == model.MovEstorno {
		return model.Movement{}, fmt.Errorf("%w: um estorno não pode ser estornado", ErrInvalid)
	}
	if orig.Estornado {
		return model.Movement{}, fmt.Errorf("%w: %s já foi estornado por %s", ErrConflict, id, orig.EstornadoPor)
	}
	changes := make([]model.StockChange, 0, len(orig.Itens))
	for _, it := range orig.Itens {
		changes = append(changes, model.StockChange{SKU: it.SKU, Delta: -it.Movimentado})
	}
	rev, err := s.apply(model.Movement{
		Tipo: model.MovEstorno, MachineID: orig.MachineID, EstornoDe: id,
		NumerosSerie: orig.NumerosSerie, Observacao: observacao,
	}, changes)
	if err != nil {
		return model.Movement{}, err
	}
	s.d.Movements[i].Estornado = true
	s.d.Movements[i].EstornadoPor = rev.ID
	return rev, s.save()
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// Arruma os dados da peça antes de salvar: tira espaços sobrando, descobre
// o tipo pela unidade (rolo/m = consumível) e limpa fornecedores vazios.
func normalize(p *model.Part) {
	p.Nome = strings.TrimSpace(p.Nome)
	p.FornecedorLegado = ""
	fs := []model.Fornecimento{}
	for _, f := range p.Fornecedores {
		f.Nome, f.Codigo = strings.TrimSpace(f.Nome), strings.TrimSpace(f.Codigo)
		if f.Nome != "" || f.Codigo != "" {
			fs = append(fs, f)
		}
	}
	p.Fornecedores = fs
	if p.Unidade == "" {
		p.Unidade = "und"
	}
	if p.Tipo == "" {
		switch p.Unidade {
		case "rolo", "m":
			p.Tipo = "consumivel"
		case "serviço":
			p.Tipo = "servico"
		default:
			p.Tipo = "peca"
		}
	}
}
