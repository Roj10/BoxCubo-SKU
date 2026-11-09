// ============================================================
//  ESTOQUE MÍNIMO E REPOSIÇÃO
// ============================================================
//  Vocês definem, por máquina, "manter peças para N máquinas".
//  O mínimo de cada peça sai da lista de materiais:
//      mínimo da peça = quantidade por máquina x N
//  Ex.: V2 com N = 5 -> balança (1 por máquina) mínimo 5;
//       sensor ultrassônico (10 por máquina) mínimo 50.
//  Se a mesma peça é usada em mais de um modelo, os mínimos se SOMAM.

package api

import (
	"math"
	"net/http"
	"sort"

	"boxcubo-sku/internal/model"
)

func (h *handler) routesMinimo(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/machines/{id}/minimo", h.setMinimo)
	mux.HandleFunc("GET /api/reposicao", h.reposicao)
}

// minimos calcula o estoque mínimo de cada peça: para cada máquina com
// mínimo configurado, quantidade no BOM × número de máquinas. Peças usadas em
// mais de um modelo somam os mínimos.
// Calcula o mínimo de todas as peças, passando por cada máquina que tem
// mínimo configurado.
func (h *handler) minimos() (min map[string]float64, porMaquina map[string][]string) {
	min = map[string]float64{}
	porMaquina = map[string][]string{}
	// soma "qtd" de "sku" no mínimo; se for subconjunto (ex.: CPU), o mínimo
	// vai para as peças da receita dele (Raspberry, display, Shield -> resistores...),
	// porque o que se compra são as peças, e o subconjunto se fabrica com elas.
	var soma func(sku string, qtd float64, nome string, profundidade int)
	soma = func(sku string, qtd float64, nome string, profundidade int) {
		if qtd <= 0 || profundidade > 10 {
			return
		}
		p, err := h.s.GetPart(sku)
		if err != nil {
			return
		}
		if p.Fabricado {
			receita, _ := h.s.GetBOM(sku)
			if len(receita) > 0 {
				for _, it := range receita {
					soma(it.SKU, qtd*it.Quantidade, nome, profundidade+1)
				}
				return
			}
		}
		min[sku] = round2(min[sku] + qtd)
		if !contem(porMaquina[sku], nome) {
			porMaquina[sku] = append(porMaquina[sku], nome)
		}
	}
	for _, m := range h.s.ListMachines() {
		if m.MinimoMaquinas <= 0 {
			continue
		}
		items, err := h.s.GetBOM(m.ID)
		if err != nil {
			continue
		}
		for _, it := range items {
			if it.Kit != "" && !m.MinimoComKits {
				continue
			}
			soma(it.SKU, it.Quantidade*float64(m.MinimoMaquinas), m.Nome, 0)
		}
	}
	return min, porMaquina
}

// Lista as peças com estoque MENOR que o mínimo e quanto comprar para
// voltar a ele (sugestão = mínimo - estoque). Zeradas aparecem primeiro.
func (h *handler) abaixoDoMinimo() []model.ReposicaoLine {
	min, porMaquina := h.minimos()
	out := []model.ReposicaoLine{}
	for sku, m := range min {
		p, err := h.s.GetPart(sku)
		if err != nil || p.Estoque >= m {
			continue
		}
		sug := m - p.Estoque
		if p.Tipo == "peca" || p.Tipo == "servico" {
			sug = math.Ceil(sug - 1e-9)
		} else {
			sug = round2(sug)
		}
		l := model.ReposicaoLine{
			SKU: p.SKU, Nome: p.Nome, Unidade: p.Unidade, Estoque: p.Estoque, Minimo: m,
			Sugestao: sug, ValorUnit: p.ValorUnitario, Fornecedores: p.Fornecedores,
			Imagem: p.Imagem, Zerado: p.Estoque <= 0, Maquinas: porMaquina[sku],
		}
		if p.ValorUnitario != nil {
			c := round2(sug * *p.ValorUnitario)
			l.Custo = &c
		}
		out = append(out, l)
	}
	// zerados primeiro, depois pelo SKU
	sort.Slice(out, func(i, j int) bool {
		if out[i].Zerado != out[j].Zerado {
			return out[i].Zerado
		}
		return out[i].SKU < out[j].SKU
	})
	return out
}

type reposicaoResp struct {
	Linhas     []model.ReposicaoLine `json:"linhas"`
	Itens      int                   `json:"itens"`
	Zerados    int                   `json:"zerados"`
	CustoTotal float64               `json:"custo_total"`
	SemPreco   int                   `json:"sem_preco"`
	Maquinas   []model.Machine       `json:"maquinas"` // máquinas com mínimo configurado
}

// Tela "Reposição": a lista acima + totais (quantas peças, quantas zeradas,
// custo para repor). Também é usada pelo número laranja da barra lateral.
func (h *handler) reposicao(w http.ResponseWriter, r *http.Request) {
	resp := reposicaoResp{Linhas: h.abaixoDoMinimo(), Maquinas: []model.Machine{}}
	resp.Itens = len(resp.Linhas)
	for _, l := range resp.Linhas {
		if l.Zerado {
			resp.Zerados++
		}
		if l.Custo != nil {
			resp.CustoTotal += *l.Custo
		} else {
			resp.SemPreco++
		}
	}
	resp.CustoTotal = round2(resp.CustoTotal)
	for _, m := range h.s.ListMachines() {
		if m.MinimoMaquinas > 0 {
			resp.Maquinas = append(resp.Maquinas, m)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// Botão "Configurar/Alterar" da faixa "Estoque mínimo" na página da máquina.
func (h *handler) setMinimo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Maquinas int  `json:"maquinas"`
		ComKits  bool `json:"com_kits"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	m, err := h.s.SetMinimo(r.PathValue("id"), req.Maquinas, req.ComKits)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// movResp é a resposta de uma saída: a movimentação + o que ficou abaixo do mínimo.
type movResp struct {
	model.Movement
	AbaixoMinimo []model.ReposicaoLine `json:"abaixo_minimo"`
}

// comAlertaMinimo devolve só as peças desta movimentação que ficaram abaixo do mínimo.
// Depois de uma saída, separa só as peças DESSA saída que ficaram abaixo
// do mínimo (é a caixa amarela na confirmação da saída).
func (h *handler) comAlertaMinimo(mov model.Movement) movResp {
	mexidas := map[string]bool{}
	for _, it := range mov.Itens {
		mexidas[it.SKU] = true
	}
	alerta := []model.ReposicaoLine{}
	for _, l := range h.abaixoDoMinimo() {
		if mexidas[l.SKU] {
			alerta = append(alerta, l)
		}
	}
	return movResp{Movement: mov, AbaixoMinimo: alerta}
}
