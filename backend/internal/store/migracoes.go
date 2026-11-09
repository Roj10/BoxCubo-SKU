// ============================================================
//  MIGRAÇÕES: atualizações automáticas de dados antigos
// ============================================================
//  Quando o sistema muda o jeito de guardar alguma coisa (ex.: os SKUs
//  perderam o "BC-"), os dados que já existem no computador de vocês
//  precisam ser convertidos. Cada conversão aqui é uma "migração".
//
//  Funciona como um checklist: cada migração tem um nome; depois de rodar,
//  o nome fica anotado no arquivo de dados ("migracoes") e ela nunca mais
//  roda de novo. Elas rodam sozinhas toda vez que o backend liga.

package store

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"boxcubo-sku/internal/model"
)

//go:embed sku_descritivo.json
var skuDescritivoJSON []byte

// migracao altera o arquivo de dados uma única vez (o nome fica gravado em
// "migracoes"). Rodam em ordem, com o lock já pego.
type migracao struct {
	nome   string
	aplica func(s *FileStore, imgDir string) (string, error)
}

// A lista, em ordem, de todas as conversões que o sistema já teve.
var migracoes = []migracao{
	{"sku-descritivo-v1", migrarSKUDescritivo},
	{"sem-bc-v1", migrarSemBC},
	{"fornecedores-v1", migrarFornecedores},
	{"receitas-controle-v1", migrarReceitasControle},
	{"remover-maquinas-v1", migrarRemoverMaquinas},
}

// Migrar aplica as migrações que ainda não rodaram neste arquivo e devolve
// uma frase para cada uma que mudou alguma coisa.
// Roda as migrações que ainda não rodaram e devolve uma frase para cada
// uma que mudou algo (é o que aparece no terminal: "Dados atualizados — ...").
func (s *FileStore) Migrar(imgDir string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	feitas := map[string]bool{}
	for _, m := range s.d.Migracoes {
		feitas[m] = true
	}
	msgs := []string{}
	mudou := false
	for _, m := range migracoes {
		if feitas[m.nome] {
			continue
		}
		msg, err := m.aplica(s, imgDir)
		if err != nil {
			return msgs, fmt.Errorf("migração %s: %w", m.nome, err)
		}
		if msg != "" {
			msgs = append(msgs, msg)
		}
		s.d.Migracoes = append(s.d.Migracoes, m.nome)
		mudou = true
	}
	if mudou {
		return msgs, s.save()
	}
	return msgs, nil
}

// SKUs numerados da V2 -> finais descritivos (BC-ELE-RES-001 -> BC-ELE-RES-10).
// Migração 1: troca os SKUs numerados pelos descritivos usando a tabela
// sku_descritivo.json (ex.: BC-ELE-RES-001 -> BC-ELE-RES-10).
func migrarSKUDescritivo(s *FileStore, imgDir string) (string, error) {
	var pares [][2]string
	if err := json.Unmarshal(skuDescritivoJSON, &pares); err != nil {
		return "", err
	}
	n := 0
	for _, par := range pares {
		antigo, novo := par[0], par[1]
		if s.partIndex(antigo) < 0 || s.partIndex(novo) >= 0 {
			continue // já migrado, apagado ou com conflito: não mexe
		}
		if _, err := s.renameSKU(antigo, novo); err != nil {
			return "", err
		}
		RenomearArquivosImagem(imgDir, antigo, novo)
		n++
	}
	if n == 0 {
		return "", nil
	}
	return fmt.Sprintf("SKUs com final descritivo: %d peças", n), nil
}

// Tira o "BC-" do começo dos SKUs (BC-ELE-RES-10 -> ELE-RES-10, BC-MAQ-V2 -> MAQ-V2).
// Migração 2: tira o "BC-" do começo dos códigos
// (BC-ELE-RES-10 -> ELE-RES-10 e BC-MAQ-V2 -> MAQ-V2).
func migrarSemBC(s *FileStore, imgDir string) (string, error) {
	n := 0
	for _, p := range append([]model.Part(nil), s.d.Parts...) {
		if !strings.HasPrefix(p.SKU, "BC-") {
			continue
		}
		novo := strings.TrimPrefix(p.SKU, "BC-")
		if s.partIndex(novo) >= 0 {
			continue
		}
		if _, err := s.renameSKU(p.SKU, novo); err != nil {
			return "", err
		}
		RenomearArquivosImagem(imgDir, p.SKU, novo)
		n++
	}
	for i := range s.d.Machines {
		s.d.Machines[i].SKU = strings.TrimPrefix(s.d.Machines[i].SKU, "BC-")
	}
	if n == 0 {
		return "", nil
	}
	return fmt.Sprintf("SKUs sem o \"BC-\": %d peças", n), nil
}

// Converte o texto livre "FORNELL / ML" numa lista de fornecedores com código.
// Migração 3: o fornecedor era um texto solto ("FORNELL / ML"); agora vira
// uma lista organizada, com espaço para o código do item em cada um.
func migrarFornecedores(s *FileStore, _ string) (string, error) {
	n := 0
	for i := range s.d.Parts {
		p := &s.d.Parts[i]
		if len(p.Fornecedores) == 0 && strings.TrimSpace(p.FornecedorLegado) != "" {
			p.Fornecedores = FornecedoresDeTexto(p.FornecedorLegado, p.Nome)
			n++
		}
		p.FornecedorLegado = ""
		if p.Fornecedores == nil {
			p.Fornecedores = []model.Fornecimento{}
		}
	}
	if n == 0 {
		return "", nil
	}
	return fmt.Sprintf("Fornecedores separados com código: %d peças", n), nil
}

// nomes padronizados dos fornecedores que aparecem na planilha
// Tabela para padronizar os nomes que vinham escritos de jeitos diferentes
// na planilha (ex.: "ML" -> "Mercado Livre", "FORNELL" -> "Fornell").
var nomesFornecedor = map[string]string{
	"fornell":                 "Fornell",
	"ml":                      "Mercado Livre",
	"mercado livre":           "Mercado Livre",
	"1001 parafusos":          "1001 Parafusos",
	"r&a parafusos":           "R&A Parafusos",
	"abu mazin":               "Abu Mazin",
	"wincabos":                "Wincabos",
	"griffus":                 "Griffus",
	"trentim":                 "Trentim",
	"dominik":                 "Dominik",
	"gr":                      "GR",
	"metalporto":              "MetalPorto",
	"zinca rapido":            "Zinca Rápido",
	"tecnograph":              "Tecnograph",
	"sombra print":            "Sombra Print",
	"patola":                  "Patola",
	"trombini":                "Trombini",
	"lri mw":                  "LRI MW",
	"kabum":                   "Kabum",
	"bite":                    "Bite",
	"loja completa":           "Loja Completa",
	"casa das borrachas":      "Casa das Borrachas",
	"mercadão das embalagens": "Mercadão das Embalagens",
	"cia embalagem":           "Cia Embalagem",
}

var (
	reURL          = regexp.MustCompile(`:?\s*https?://\S+`)
	reCodigoInicio = regexp.MustCompile(`^(\d{3,6})\s`) // "2241 Parafuso Frances..." (1001 Parafusos)
	reCodigoParen  = regexp.MustCompile(`\((\d{5,})\)`) // "Caixa de papelão (398737)" (Trombini)
)

// FornecedoresDeTexto separa "Dominik / MetalPorto" em itens e tenta achar o
// código do item no fornecedor dentro do nome da peça.
// Quebra o texto antigo em fornecedores e tenta achar o código do item
// escondido no nome da peça. Ex.: na peça "2241 Parafuso Frances...", o
// "2241" é o código da 1001 Parafusos; na "Caixa de papelão (398737)",
// o "398737" é o código da Trombini.
func FornecedoresDeTexto(txt, nomePeca string) []model.Fornecimento {
	txt = reURL.ReplaceAllString(txt, "")
	out := []model.Fornecimento{}
	vistos := map[string]bool{}
	for _, parte := range strings.Split(txt, "/") {
		nome := strings.TrimSpace(parte)
		if nome == "" {
			continue
		}
		if padrao, ok := nomesFornecedor[strings.ToLower(nome)]; ok {
			nome = padrao
		}
		if vistos[nome] {
			continue
		}
		vistos[nome] = true
		f := model.Fornecimento{Nome: nome}
		switch nome {
		case "1001 Parafusos":
			if m := reCodigoInicio.FindStringSubmatch(nomePeca); m != nil {
				f.Codigo = m[1]
			}
		case "Trombini":
			if m := reCodigoParen.FindStringSubmatch(nomePeca); m != nil {
				f.Codigo = m[1]
			}
		}
		out = append(out, f)
	}
	return out
}

//go:embed receitas_controle.json
var receitasControleJSON []byte

type materialControle struct {
	NomeControle string   `json:"nome_controle"`
	SKU          string   `json:"sku"`
	Novo         bool     `json:"novo"`
	Nome         string   `json:"nome"`
	Grupo        string   `json:"grupo"`
	Classe       string   `json:"classe"`
	Unidade      string   `json:"unidade"`
	TrocaUnidade bool     `json:"troca_unidade"`
	Tipo         string   `json:"tipo"`
	Estoque      float64  `json:"estoque"`
	Preco        *float64 `json:"preco"`
	Fabricado    bool     `json:"fabricado"`
}

// Migração 4: traz do sistema Controle BoxCubo (aba Materiais) as receitas da
// V2, Vision e EYE e dos subconjuntos CPU e Shield Raspberry.
//   - peças que já existiam: reaproveita o SKU (atualiza preço quando o
//     Controle tem e marca CPU/Shield como fabricados);
//   - peças novas: cria com SKU novo (ex.: EST-MET-EYEBRACO1);
//   - a receita da V2 passa a ser a do Controle (os itens do kit Correios
//     continuam como opcionais);
//   - o estoque passa a ser o do Controle, lançado como uma movimentação de
//     AJUSTE (aparece no histórico e pode ser estornada).
func migrarReceitasControle(s *FileStore, _ string) (string, error) {
	var dados struct {
		Data      string                       `json:"data"`
		Materiais []materialControle           `json:"materiais"`
		Raw       map[string][]json.RawMessage `json:"receitas"`
	}
	if err := json.Unmarshal(receitasControleJSON, &dados); err != nil {
		return "", err
	}
	criados := 0
	ajustes := []model.StockChange{}
	for _, m := range dados.Materiais {
		i := s.partIndex(m.SKU)
		if i < 0 {
			p := model.Part{
				SKU: m.SKU, Nome: m.Nome, Grupo: m.Grupo, Classe: m.Classe, Unidade: m.Unidade,
				Tipo: m.Tipo, ValorUnitario: m.Preco, Fabricado: m.Fabricado,
				Fornecedores: []model.Fornecimento{},
			}
			normalize(&p)
			s.d.Parts = append(s.d.Parts, p)
			i = len(s.d.Parts) - 1
			criados++
		}
		p := &s.d.Parts[i]
		p.Nome = m.Nome // usa o nome do Controle, que é o que a equipe usa hoje
		if m.Preco != nil {
			p.ValorUnitario = m.Preco
		}
		if m.Fabricado {
			p.Fabricado = true
		}
		if m.TrocaUnidade && p.Unidade != m.Unidade {
			// ex.: cabo manga era contado em rolo, no Controle é em metro
			p.Unidade, p.Tipo, p.Estoque = m.Unidade, m.Tipo, 0
		}
		if d := round3(m.Estoque - p.Estoque); d != 0 {
			ajustes = append(ajustes, model.StockChange{SKU: m.SKU, Delta: d})
		}
	}

	// receitas: [[sku, quantidade], ...]
	receita := func(dono string) ([]model.BOMItem, error) {
		out := []model.BOMItem{}
		for _, raw := range dados.Raw[dono] {
			var par [2]any
			if err := json.Unmarshal(raw, &par); err != nil {
				return nil, err
			}
			sku, _ := par[0].(string)
			q, _ := par[1].(float64)
			out = append(out, model.BOMItem{SKU: sku, Quantidade: q, OndeUsado: []string{}})
		}
		return out, nil
	}
	for _, dono := range []string{"vision", "eye", "ELE-SUB-CPUV2", "ELE-PCB-SHIELDRPI"} {
		lista, err := receita(dono)
		if err != nil {
			return "", err
		}
		s.d.BOM[dono] = lista
	}
	// V2: receita do Controle + mantém "onde é usado" e os itens de kit que já existiam
	novaV2, err := receita("v2")
	if err != nil {
		return "", err
	}
	antiga := map[string]model.BOMItem{}
	for _, it := range s.d.BOM["v2"] {
		antiga[it.SKU] = it
	}
	naNova := map[string]bool{}
	for j, it := range novaV2 {
		naNova[it.SKU] = true
		if a, ok := antiga[it.SKU]; ok {
			novaV2[j].OndeUsado = a.OndeUsado
		}
	}
	for _, it := range s.d.BOM["v2"] {
		if it.Kit != "" && !naNova[it.SKU] {
			novaV2 = append(novaV2, it)
		}
	}
	s.d.BOM["v2"] = novaV2

	// máquinas: Vision e Eye ganham receita; a Slim entra na lista (ainda sem receita)
	temSlim := false
	for i := range s.d.Machines {
		switch s.d.Machines[i].ID {
		case "vision", "eye":
			s.d.Machines[i].Ativa = true
		case "slim":
			temSlim = true
		}
	}
	if !temSlim {
		s.d.Machines = append(s.d.Machines, model.Machine{
			ID: "slim", SKU: "MAQ-SLIM", Nome: "BoxCubo Slim", Apelido: "Compacta",
			Tecnologia: "A definir", Ativa: false,
		})
	}

	if len(ajustes) > 0 {
		if _, err := s.apply(model.Movement{
			Tipo:       model.MovAjuste,
			Observacao: "Estoque importado do Controle BoxCubo (192.168.15.10) em " + dados.Data,
		}, ajustes); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("Receitas do Controle BoxCubo: V2, Vision, EYE, CPU e Shield (%d peças novas, estoque de %d peças ajustado)",
		criados, len(ajustes)), nil
}

// Migração 5: tira do sistema as máquinas que a BoxCubo não usa mais
// (Estática V2.5, Dynamic e Slim), junto com a lista de peças delas, se
// tiver. As PEÇAS continuam cadastradas (algumas são usadas em outras
// máquinas, como a "PCB BoxCubo v2.5" da V2) e o histórico de
// movimentações não é apagado.
func migrarRemoverMaquinas(s *FileStore, _ string) (string, error) {
	tirar := map[string]bool{"v25": true, "dynamic": true, "slim": true}
	ficam := s.d.Machines[:0]
	nomes := []string{}
	for _, m := range s.d.Machines {
		if tirar[m.ID] {
			nomes = append(nomes, m.Nome)
			delete(s.d.BOM, m.ID)
			continue
		}
		ficam = append(ficam, m)
	}
	s.d.Machines = ficam
	if len(nomes) == 0 {
		return "", nil
	}
	return "máquinas removidas: " + strings.Join(nomes, ", "), nil
}
