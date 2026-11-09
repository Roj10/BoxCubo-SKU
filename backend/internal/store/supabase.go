// ============================================================
//  SUPABASE: os dados guardados num banco de dados na nuvem
// ============================================================
//  Até aqui tudo ficava num arquivo (data/boxcubo.json). Com o Supabase,
//  os dados ficam em tabelas de um banco Postgres na nuvem:
//
//    maquinas, kits, pecas, pecas_fornecedores, receitas, legenda,
//    movimentacoes, migracoes
//
//  Como funciona:
//   - quando o backend liga, ele lê todas as tabelas e monta os dados na
//     memória (igual fazia com o arquivo);
//   - cada vez que algo muda, ele compara com o que já estava no banco e
//     manda SÓ o que mudou (linhas novas/alteradas e linhas apagadas);
//   - na primeira vez, com o banco vazio, ele copia para o Supabase os
//     dados do arquivo local (data/boxcubo.json), se existir.
//
//  A conversa com o Supabase é pela "API REST" dele (endereços https://...
//  /rest/v1/<tabela>), usando a CHAVE SECRETA do projeto. A chave secreta
//  fica só no backend (arquivo .env), nunca na tela nem no GitHub.
//
//  Importante: o backend deve ser o único programa escrevendo nessas
//  tabelas (como era com o arquivo). Editar direto pelo painel do Supabase
//  com o backend ligado pode ser desfeito pela próxima gravação dele.

package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"boxcubo-sku/internal/model"
)

// ---------- as linhas de cada tabela ----------

type linhaMaquina struct {
	ID             string `json:"id"`
	SKU            string `json:"sku"`
	Nome           string `json:"nome"`
	Apelido        string `json:"apelido"`
	Tecnologia     string `json:"tecnologia"`
	Ativa          bool   `json:"ativa"`
	MinimoMaquinas int    `json:"minimo_maquinas"`
	MinimoComKits  bool   `json:"minimo_com_kits"`
	Ordem          int    `json:"ordem"`
}

type linhaKit struct {
	ID        string `json:"id"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Ordem     int    `json:"ordem"`
}

type linhaPeca struct {
	SKU           string   `json:"sku"`
	Nome          string   `json:"nome"`
	Grupo         string   `json:"grupo"`
	Classe        string   `json:"classe"`
	Unidade       string   `json:"unidade"`
	Tipo          string   `json:"tipo"`
	ValorUnitario *float64 `json:"valor_unitario"`
	Estoque       float64  `json:"estoque"`
	Imagem        string   `json:"imagem"`
	Fabricado     bool     `json:"fabricado"`
	Ordem         int      `json:"ordem"`
}

type linhaFornecedor struct {
	SKU    string `json:"sku"`
	Ordem  int    `json:"ordem"` // 0 = fornecedor principal
	Nome   string `json:"nome"`
	Codigo string `json:"codigo"`
}

type linhaReceita struct {
	Dono       string   `json:"dono"` // máquina (v2, eye...) ou subconjunto (ELE-SUB-CPUV2)
	Ordem      int      `json:"ordem"`
	SKU        string   `json:"sku"`
	Quantidade float64  `json:"quantidade"`
	OndeUsado  []string `json:"onde_usado"`
	Kit        string   `json:"kit"`
}

type linhaLegenda struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
}

type linhaMovimentacao struct {
	ID           string               `json:"id"`
	Data         string               `json:"data"`
	Tipo         string               `json:"tipo"`
	MachineID    string               `json:"machine_id"`
	Quantidade   int                  `json:"quantidade"`
	NumerosSerie []string             `json:"numeros_serie"`
	Kits         []string             `json:"kits"`
	Cliente      string               `json:"cliente"`
	Observacao   string               `json:"observacao"`
	ComFalta     bool                 `json:"com_falta"`
	Estornado    bool                 `json:"estornado"`
	EstornadoPor string               `json:"estornado_por"`
	EstornoDe    string               `json:"estorno_de"`
	Itens        []model.MovementItem `json:"itens"`
}

type linhaMigracao struct {
	Nome  string `json:"nome"`
	Ordem int    `json:"ordem"`
}

// Cada tabela: nome, colunas que formam a "chave" (o que identifica a
// linha) e a ordem de leitura.
type tabela struct {
	nome  string
	chave []string
	ordem string
}

var tabelas = []tabela{
	{"maquinas", []string{"id"}, "ordem"},
	{"kits", []string{"id"}, "ordem"},
	{"pecas", []string{"sku"}, "ordem"},
	{"pecas_fornecedores", []string{"sku", "ordem"}, "sku,ordem"},
	{"receitas", []string{"dono", "ordem"}, "dono,ordem"},
	{"legenda", []string{"codigo"}, "codigo"},
	{"movimentacoes", []string{"id"}, "id"},
	{"migracoes", []string{"nome"}, "ordem"},
}

// foto = cada tabela como "chave -> linha em JSON". Comparando a foto de
// antes com a de agora, dá para saber o que mudou.
type foto map[string]map[string]string

// linhasDe transforma os dados da memória em linhas das tabelas.
func linhasDe(d *data) map[string][]any {
	t := map[string][]any{}
	for i, m := range d.Machines {
		t["maquinas"] = append(t["maquinas"], linhaMaquina{m.ID, m.SKU, m.Nome, m.Apelido, m.Tecnologia, m.Ativa, m.MinimoMaquinas, m.MinimoComKits, i})
	}
	for i, k := range d.Kits {
		t["kits"] = append(t["kits"], linhaKit{k.ID, k.Nome, k.Descricao, i})
	}
	for i, p := range d.Parts {
		t["pecas"] = append(t["pecas"], linhaPeca{p.SKU, p.Nome, p.Grupo, p.Classe, p.Unidade, p.Tipo, p.ValorUnitario, p.Estoque, p.Imagem, p.Fabricado, i})
		for j, f := range p.Fornecedores {
			t["pecas_fornecedores"] = append(t["pecas_fornecedores"], linhaFornecedor{p.SKU, j, f.Nome, f.Codigo})
		}
	}
	for dono, itens := range d.BOM {
		for i, it := range itens {
			onde := it.OndeUsado
			if onde == nil {
				onde = []string{}
			}
			t["receitas"] = append(t["receitas"], linhaReceita{dono, i, it.SKU, it.Quantidade, onde, it.Kit})
		}
	}
	for c, desc := range d.Legenda {
		t["legenda"] = append(t["legenda"], linhaLegenda{c, desc})
	}
	for _, m := range d.Movements {
		ns, kits := m.NumerosSerie, m.Kits
		if ns == nil {
			ns = []string{}
		}
		if kits == nil {
			kits = []string{}
		}
		itens := m.Itens
		if itens == nil {
			itens = []model.MovementItem{}
		}
		t["movimentacoes"] = append(t["movimentacoes"], linhaMovimentacao{
			m.ID, m.Data, m.Tipo, m.MachineID, m.Quantidade, ns, kits, m.Cliente, m.Observacao,
			m.ComFalta, m.Estornado, m.EstornadoPor, m.EstornoDe, itens,
		})
	}
	for i, n := range d.Migracoes {
		t["migracoes"] = append(t["migracoes"], linhaMigracao{n, i})
	}
	return t
}

// tirarFoto monta a foto (chave -> JSON) de todas as tabelas.
func tirarFoto(d *data) (foto, error) {
	f := foto{}
	linhas := linhasDe(d)
	for _, tb := range tabelas {
		f[tb.nome] = map[string]string{}
		for _, l := range linhas[tb.nome] {
			raw, err := json.Marshal(l)
			if err != nil {
				return nil, err
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			f[tb.nome][chaveDe(tb, m)] = string(raw)
		}
	}
	return f, nil
}

// chaveDe junta os valores das colunas-chave: "ELE-RES-10" ou "v2\x000".
func chaveDe(tb tabela, linha map[string]any) string {
	partes := make([]string, len(tb.chave))
	for i, c := range tb.chave {
		partes[i] = valorTexto(linha[c])
	}
	return strings.Join(partes, "\x00")
}

func valorTexto(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}

// ---------- conversa com o Supabase ----------

// supabase guarda o endereço, a chave e a última foto gravada no banco.
type supabase struct {
	url    string // https://xxxx.supabase.co
	chave  string // chave secreta (sb_secret_...) ou service_role
	http   *http.Client
	gravou foto // o que já está no banco
}

// ErrTabelasFaltando: o banco ainda não tem as tabelas (falta rodar o SQL).
var ErrTabelasFaltando = errors.New("as tabelas ainda não existem no Supabase: rode o arquivo backend/supabase/schema.sql no SQL Editor do Supabase")

func (sb *supabase) pedido(metodo, caminho string, corpo any, extra map[string]string) ([]byte, http.Header, error) {
	var r io.Reader
	if corpo != nil {
		raw, err := json.Marshal(corpo)
		if err != nil {
			return nil, nil, err
		}
		r = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(metodo, sb.url+"/rest/v1/"+caminho, r)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("apikey", sb.chave)
	// as chaves antigas (JWT, começam com "eyJ") também vão no Authorization
	if strings.HasPrefix(sb.chave, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+sb.chave)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := sb.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("sem conexão com o Supabase: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if resp.StatusCode == 404 || strings.Contains(msg, "PGRST205") || strings.Contains(msg, "42P01") {
			return nil, nil, ErrTabelasFaltando
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, nil, fmt.Errorf("o Supabase recusou a chave (%d): use a chave SECRETA (sb_secret_...) em SUPABASE_SECRET_KEY: %s", resp.StatusCode, msg)
		}
		return nil, nil, fmt.Errorf("Supabase respondeu %d em %s %s: %s", resp.StatusCode, metodo, caminho, msg)
	}
	return body, resp.Header, nil
}

// ler busca todas as linhas de uma tabela (de 1000 em 1000).
func (sb *supabase) ler(tb tabela) ([]json.RawMessage, error) {
	todas := []json.RawMessage{}
	for inicio := 0; ; inicio += 1000 {
		q := url.Values{"select": {"*"}, "order": {tb.ordem}, "limit": {"1000"}, "offset": {strconv.Itoa(inicio)}}
		body, _, err := sb.pedido("GET", tb.nome+"?"+q.Encode(), nil, nil)
		if err != nil {
			return nil, err
		}
		var pag []json.RawMessage
		if err := json.Unmarshal(body, &pag); err != nil {
			return nil, fmt.Errorf("lendo %s: %w", tb.nome, err)
		}
		todas = append(todas, pag...)
		if len(pag) < 1000 {
			return todas, nil
		}
	}
}

// carregar lê o banco inteiro e monta os dados. vazio = banco sem nada.
func (sb *supabase) carregar() (d data, vazio bool, err error) {
	brutas := map[string][]json.RawMessage{}
	total := 0
	for _, tb := range tabelas {
		l, err := sb.ler(tb)
		if err != nil {
			return d, false, err
		}
		brutas[tb.nome] = l
		total += len(l)
	}
	if total == 0 {
		return d, true, nil
	}
	ler := func(nome string, destino any) error {
		raw, _ := json.Marshal(brutas[nome])
		if err := json.Unmarshal(raw, destino); err != nil {
			return fmt.Errorf("lendo %s: %w", nome, err)
		}
		return nil
	}
	var (
		maqs  []linhaMaquina
		kits  []linhaKit
		pecas []linhaPeca
		forns []linhaFornecedor
		recs  []linhaReceita
		legs  []linhaLegenda
		movs  []linhaMovimentacao
		migs  []linhaMigracao
	)
	for _, x := range []struct {
		n string
		v any
	}{{"maquinas", &maqs}, {"kits", &kits}, {"pecas", &pecas}, {"pecas_fornecedores", &forns},
		{"receitas", &recs}, {"legenda", &legs}, {"movimentacoes", &movs}, {"migracoes", &migs}} {
		if err := ler(x.n, x.v); err != nil {
			return d, false, err
		}
	}

	d.Machines = []model.Machine{}
	for _, m := range maqs {
		d.Machines = append(d.Machines, model.Machine{ID: m.ID, SKU: m.SKU, Nome: m.Nome, Apelido: m.Apelido,
			Tecnologia: m.Tecnologia, Ativa: m.Ativa, MinimoMaquinas: m.MinimoMaquinas, MinimoComKits: m.MinimoComKits})
	}
	d.Kits = []model.Kit{}
	for _, k := range kits {
		d.Kits = append(d.Kits, model.Kit{ID: k.ID, Nome: k.Nome, Descricao: k.Descricao})
	}
	fornPorSKU := map[string][]linhaFornecedor{}
	for _, f := range forns {
		fornPorSKU[f.SKU] = append(fornPorSKU[f.SKU], f)
	}
	d.Parts = []model.Part{}
	for _, p := range pecas {
		fs := fornPorSKU[p.SKU]
		sort.Slice(fs, func(i, j int) bool { return fs[i].Ordem < fs[j].Ordem })
		part := model.Part{SKU: p.SKU, Nome: p.Nome, Grupo: p.Grupo, Classe: p.Classe, Unidade: p.Unidade, Tipo: p.Tipo,
			ValorUnitario: p.ValorUnitario, Estoque: p.Estoque, Imagem: p.Imagem, Fabricado: p.Fabricado,
			Fornecedores: []model.Fornecimento{}}
		for _, f := range fs {
			part.Fornecedores = append(part.Fornecedores, model.Fornecimento{Nome: f.Nome, Codigo: f.Codigo})
		}
		d.Parts = append(d.Parts, part)
	}
	d.BOM = map[string][]model.BOMItem{}
	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].Dono != recs[j].Dono {
			return recs[i].Dono < recs[j].Dono
		}
		return recs[i].Ordem < recs[j].Ordem
	})
	for _, r := range recs {
		onde := r.OndeUsado
		if onde == nil {
			onde = []string{}
		}
		d.BOM[r.Dono] = append(d.BOM[r.Dono], model.BOMItem{SKU: r.SKU, Quantidade: r.Quantidade, OndeUsado: onde, Kit: r.Kit})
	}
	d.Legenda = map[string]string{}
	for _, l := range legs {
		d.Legenda[l.Codigo] = l.Descricao
	}
	d.Movements = []model.Movement{}
	for _, m := range movs {
		mv := model.Movement{ID: m.ID, Data: m.Data, Tipo: m.Tipo, MachineID: m.MachineID, Quantidade: m.Quantidade,
			Cliente: m.Cliente, Observacao: m.Observacao, ComFalta: m.ComFalta, Estornado: m.Estornado,
			EstornadoPor: m.EstornadoPor, EstornoDe: m.EstornoDe, Itens: m.Itens}
		if len(m.NumerosSerie) > 0 {
			mv.NumerosSerie = m.NumerosSerie
		}
		if len(m.Kits) > 0 {
			mv.Kits = m.Kits
		}
		if mv.Itens == nil {
			mv.Itens = []model.MovementItem{}
		}
		d.Movements = append(d.Movements, mv)
	}
	d.Migracoes = []string{}
	for _, m := range migs {
		d.Migracoes = append(d.Migracoes, m.Nome)
	}
	return d, false, nil
}

// salvar manda para o banco só o que mudou desde a última gravação.
func (sb *supabase) salvar(d *data) error {
	agora, err := tirarFoto(d)
	if err != nil {
		return err
	}
	for _, tb := range tabelas {
		antes, novo := sb.gravou[tb.nome], agora[tb.nome]
		// linhas apagadas
		apagar := []map[string]any{}
		for k, raw := range antes {
			if _, fica := novo[k]; !fica {
				var m map[string]any
				_ = json.Unmarshal([]byte(raw), &m)
				apagar = append(apagar, m)
			}
		}
		if err := sb.apagar(tb, apagar); err != nil {
			return err
		}
		// linhas novas ou alteradas
		mandar := []json.RawMessage{}
		chaves := make([]string, 0, len(novo))
		for k := range novo {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		for _, k := range chaves {
			if antes[k] != novo[k] {
				mandar = append(mandar, json.RawMessage(novo[k]))
			}
		}
		for i := 0; i < len(mandar); i += 500 {
			fim := min(i+500, len(mandar))
			_, _, err := sb.pedido("POST", tb.nome+"?on_conflict="+strings.Join(tb.chave, ","), mandar[i:fim],
				map[string]string{"Prefer": "resolution=merge-duplicates,return=minimal"})
			if err != nil {
				return err
			}
		}
		// esta tabela já está igual no banco
		sb.gravou[tb.nome] = novo
	}
	return nil
}

// apagar remove linhas, agrupando pela primeira coluna da chave para
// mandar poucos pedidos (ex.: todas as linhas da receita da v2 de uma vez).
func (sb *supabase) apagar(tb tabela, linhas []map[string]any) error {
	if len(linhas) == 0 {
		return nil
	}
	if len(tb.chave) == 1 {
		vals := []string{}
		for _, l := range linhas {
			vals = append(vals, valorTexto(l[tb.chave[0]]))
		}
		return sb.apagarEm(tb.nome, url.Values{tb.chave[0]: {emLista(vals)}})
	}
	grupos := map[string][]string{}
	for _, l := range linhas {
		a := valorTexto(l[tb.chave[0]])
		grupos[a] = append(grupos[a], valorTexto(l[tb.chave[1]]))
	}
	for a, bs := range grupos {
		if err := sb.apagarEm(tb.nome, url.Values{tb.chave[0]: {"eq." + a}, tb.chave[1]: {emLista(bs)}}); err != nil {
			return err
		}
	}
	return nil
}

func (sb *supabase) apagarEm(nome string, filtro url.Values) error {
	_, _, err := sb.pedido("DELETE", nome+"?"+filtro.Encode(), nil, map[string]string{"Prefer": "return=minimal"})
	return err
}

// emLista monta o filtro "in.(a,b,c)" do Supabase, com aspas em cada valor
// (assim vírgulas ou parênteses dentro do valor não atrapalham).
func emLista(vals []string) string {
	q := make([]string, len(vals))
	for i, v := range vals {
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		q[i] = `"` + v + `"`
	}
	return "in.(" + strings.Join(q, ",") + ")"
}

// NewSupabaseStore liga o sistema ao Supabase. Se o banco estiver vazio,
// copia para ele os dados do arquivo local (arquivoLocal) ou, se não
// houver arquivo, as peças iniciais da V2. Devolve também uma frase dizendo
// de onde vieram os dados.
func NewSupabaseStore(urlProjeto, chave, arquivoLocal string) (*FileStore, string, error) {
	sb := &supabase{
		url:    strings.TrimRight(strings.TrimSpace(urlProjeto), "/"),
		chave:  strings.TrimSpace(chave),
		http:   &http.Client{Timeout: 30 * time.Second},
		gravou: foto{},
	}
	d, vazio, err := sb.carregar()
	if err != nil {
		return nil, "", err
	}
	origem := "dados lidos do Supabase"
	if vazio {
		raw, err := os.ReadFile(arquivoLocal)
		origem = "banco vazio: dados copiados de " + arquivoLocal + " para o Supabase"
		if errors.Is(err, os.ErrNotExist) {
			raw, origem = seedJSON, "banco vazio: peças iniciais da V2 gravadas no Supabase"
		} else if err != nil {
			return nil, "", err
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, "", fmt.Errorf("lendo %s: %w", arquivoLocal, err)
		}
	} else if sb.gravou, err = tirarFoto(&d); err != nil {
		return nil, "", err
	}
	if d.BOM == nil {
		d.BOM = map[string][]model.BOMItem{}
	}
	s := &FileStore{d: d, banco: sb}
	if vazio {
		if err := s.save(); err != nil {
			return nil, "", err
		}
	}
	return s, origem, nil
}
