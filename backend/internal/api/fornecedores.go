// ============================================================
//  FORNECEDORES: a página com todos os fornecedores
// ============================================================
//  Junta, a partir das peças, quem fornece o quê:
//   - resumo: cada fornecedor, quantas peças, quantas sem código, quantas
//     estão abaixo do mínimo e quanto custa repor;
//   - peças de um fornecedor, com o código do item e a sugestão de compra;
//   - editar o código do item direto na tabela;
//   - renomear um fornecedor em todas as peças (e juntar duplicados,
//     ex.: "1001 parafusos" + "1001 Parafusos");
//   - "não trabalho mais com ele": passar as peças para outro fornecedor
//     ou só tirar ele das peças;
//   - a lista de peças que estão SEM fornecedor.
//  O pedido para um fornecedor = as peças em que ele é o PRINCIPAL e que
//  estão abaixo do estoque mínimo.

package api

import (
	"math"
	"net/http"
	"sort"
	"strings"

	"boxcubo-sku/internal/model"
)

func (h *handler) routesFornecedores(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/fornecedores/resumo", h.fornecedoresResumo)
	mux.HandleFunc("GET /api/fornecedores/pecas", h.fornecedorPecas) // ?nome=Fornell
	mux.HandleFunc("PUT /api/fornecedores/codigo", h.fornecedorCodigo)
	mux.HandleFunc("POST /api/fornecedores/renomear", h.fornecedorRenomear)
	mux.HandleFunc("POST /api/fornecedores/substituir", h.fornecedorSubstituir)
	mux.HandleFunc("GET /api/fornecedores/sem-fornecedor", h.semFornecedor)
}

// Uma peça vista pelo lado do fornecedor.
type pecaFornecedor struct {
	SKU       string   `json:"sku"`
	Nome      string   `json:"nome"`
	Unidade   string   `json:"unidade"`
	Codigo    string   `json:"codigo"`    // código do item neste fornecedor
	Principal bool     `json:"principal"` // este é o fornecedor principal da peça
	Outros    []string `json:"outros"`    // outros fornecedores da peça
	// códigos do item em todos os fornecedores da peça, ex.: "Fornell 1234"
	// (para a pesquisa achar a peça pelo código de qualquer fornecedor)
	Codigos []string `json:"codigos"`
	// "onde é usado" (ACA, ADE, CAB...) em todas as máquinas e subconjuntos
	OndeUsado []string `json:"onde_usado"`
	Estoque   float64  `json:"estoque"`
	Minimo    float64  `json:"minimo"`
	Comprar   float64  `json:"comprar"` // quanto falta para o mínimo
	ValorUnit *float64 `json:"valor_unitario"`
	Custo     *float64 `json:"custo"` // comprar x valor
	UsadaEm   []string `json:"usada_em"`
	Imagem    string   `json:"imagem"`
}

type resumoFornecedor struct {
	Nome         string  `json:"nome"`
	Pecas        int     `json:"pecas"`
	Principal    int     `json:"principal"`     // peças em que é o principal
	SemCodigo    int     `json:"sem_codigo"`    // peças sem o código do item
	AbaixoMinimo int     `json:"abaixo_minimo"` // peças (em que é principal) abaixo do mínimo
	CustoRepor   float64 `json:"custo_repor"`   // pedido sugerido para voltar ao mínimo
}

// pecasPorFornecedor monta, para cada fornecedor, a lista das peças dele.
// As peças sem nenhum fornecedor ficam na chave "" (nome vazio).
func (h *handler) pecasPorFornecedor() map[string][]pecaFornecedor {
	min, _ := h.minimos()
	usada := h.usadaEm()
	onde := h.ondeUsadoPorPeca()
	out := map[string][]pecaFornecedor{}
	for _, p := range h.s.ListParts("") {
		comprar := 0.0
		if m := min[p.SKU]; m > 0 && p.Estoque < m {
			comprar = m - p.Estoque
			if p.Tipo == "peca" || p.Tipo == "servico" {
				comprar = math.Ceil(comprar - 1e-9)
			} else {
				comprar = round2(comprar)
			}
		}
		lista := p.Fornecedores
		if len(lista) == 0 {
			lista = []model.Fornecimento{{}} // entra na chave "" (sem fornecedor)
		}
		for i, f := range lista {
			outros, codigos := []string{}, []string{}
			for j, o := range lista {
				if j != i {
					outros = append(outros, o.Nome)
				}
				if o.Codigo != "" {
					codigos = append(codigos, o.Nome+" "+o.Codigo)
				}
			}
			l := pecaFornecedor{
				SKU: p.SKU, Nome: p.Nome, Unidade: p.Unidade, Codigo: f.Codigo, Principal: i == 0 && f.Nome != "",
				Outros: outros, Codigos: codigos, OndeUsado: onde[p.SKU], Estoque: p.Estoque, Minimo: min[p.SKU], Comprar: comprar,
				ValorUnit: p.ValorUnitario, UsadaEm: usada[p.SKU], Imagem: p.Imagem,
			}
			if l.OndeUsado == nil {
				l.OndeUsado = []string{}
			}
			if l.UsadaEm == nil {
				l.UsadaEm = []string{}
			}
			if p.ValorUnitario != nil && comprar > 0 {
				c := round2(comprar * *p.ValorUnitario)
				l.Custo = &c
			}
			out[f.Nome] = append(out[f.Nome], l)
		}
	}
	for nome := range out {
		sort.Slice(out[nome], func(i, j int) bool { return out[nome][i].SKU < out[nome][j].SKU })
	}
	return out
}

func (h *handler) fornecedoresResumo(w http.ResponseWriter, r *http.Request) {
	res := []resumoFornecedor{}
	for nome, pecas := range h.pecasPorFornecedor() {
		if nome == "" {
			continue // as peças sem fornecedor têm rota própria
		}
		rf := resumoFornecedor{Nome: nome, Pecas: len(pecas)}
		for _, p := range pecas {
			if p.Codigo == "" {
				rf.SemCodigo++
			}
			if p.Principal {
				rf.Principal++
				if p.Comprar > 0 {
					rf.AbaixoMinimo++
					if p.Custo != nil {
						rf.CustoRepor += *p.Custo
					}
				}
			}
		}
		rf.CustoRepor = round2(rf.CustoRepor)
		res = append(res, rf)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Pecas != res[j].Pecas {
			return res[i].Pecas > res[j].Pecas
		}
		return res[i].Nome < res[j].Nome
	})
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) fornecedorPecas(w http.ResponseWriter, r *http.Request) {
	nome := r.URL.Query().Get("nome")
	pecas, ok := h.pecasPorFornecedor()[nome]
	if !ok || nome == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"erro": "fornecedor não encontrado: " + nome})
		return
	}
	writeJSON(w, http.StatusOK, pecas)
}

// Body: {"sku": "ELE-RES-10", "fornecedor": "Fornell", "codigo": "RES-10R"}
func (h *handler) fornecedorCodigo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SKU        string `json:"sku"`
		Fornecedor string `json:"fornecedor"`
		Codigo     string `json:"codigo"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	p, err := h.s.SetCodigoFornecedor(req.SKU, req.Fornecedor, req.Codigo)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Body: {"de": "1001 parafusos", "para": "1001 Parafusos"}
func (h *handler) fornecedorRenomear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		De   string `json:"de"`
		Para string `json:"para"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	n, err := h.s.RenomearFornecedor(req.De, req.Para)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pecas": n, "nome": strings.TrimSpace(req.Para)})
}

// Peças sem nenhum fornecedor (fabricadas na BoxCubo ficam de fora: não
// se compram).
func (h *handler) semFornecedor(w http.ResponseWriter, r *http.Request) {
	fabricadas := map[string]bool{}
	for _, p := range h.s.ListSubconjuntos() {
		fabricadas[p.SKU] = true
	}
	out := []pecaFornecedor{}
	for _, p := range h.pecasPorFornecedor()[""] {
		if !fabricadas[p.SKU] {
			out = append(out, p)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// Body: {"de": "Fornell", "para": "Mouser"} — passa as peças da Fornell
// para a Mouser. Com "para" vazio, só tira a Fornell das peças.
func (h *handler) fornecedorSubstituir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		De   string `json:"de"`
		Para string `json:"para"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	n, sem, err := h.s.SubstituirFornecedor(req.De, req.Para)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pecas": n, "sem_fornecedor": sem})
}

// ondeUsadoPorPeca junta, para cada peça, os códigos "onde é usado" (ACA,
// ADE, CAB...) de todas as listas em que ela aparece, sem repetir.
func (h *handler) ondeUsadoPorPeca() map[string][]string {
	donos := []string{}
	for _, m := range h.s.ListMachines() {
		donos = append(donos, m.ID)
	}
	for _, sub := range h.s.ListSubconjuntos() {
		donos = append(donos, sub.SKU)
	}
	out := map[string][]string{}
	for _, d := range donos {
		itens, _ := h.s.GetBOM(d)
		for _, it := range itens {
			for _, o := range it.OndeUsado {
				if !contem(out[it.SKU], o) {
					out[it.SKU] = append(out[it.SKU], o)
				}
			}
		}
	}
	for sku := range out {
		sort.Strings(out[sku])
	}
	return out
}
