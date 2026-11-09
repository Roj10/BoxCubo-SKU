// ============================================================
//  SUBCONJUNTOS: peças que a BoxCubo monta (CPU, Shield Raspberry)
// ============================================================
//  Um subconjunto é uma peça com "receita" própria. Ex.: a CPU leva
//  Raspberry, display, cartão de memória, Shield Raspberry... e o Shield
//  leva resistores, LEDs, relé...
//
//  Como funciona no estoque (igual ao Controle BoxCubo):
//   - "Registrar fabricação" de 5 CPUs: sai do estoque a receita x 5
//     e entram 5 CPUs prontas;
//   - na saída de uma máquina V2, sai 1 CPU pronta (não as peças soltas).
//
//  Nas telas, um subconjunto aparece igual a uma máquina (lista de peças,
//  planejar), por isso aqui ele é "vestido" como máquina (dono).

package api

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"boxcubo-sku/internal/model"
	"boxcubo-sku/internal/store"
)

func (h *handler) routesSubconjuntos(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/subconjuntos", h.listSubconjuntos)
	mux.HandleFunc("POST /api/parts/{sku}/fabricar", h.fabricar)
}

// dono devolve a "máquina" dona de uma lista de materiais: uma máquina de
// verdade (v2, eye...) ou um subconjunto fabricado (pelo SKU).
func (h *handler) dono(id string) (model.Machine, error) {
	if m, err := h.s.GetMachine(id); err == nil {
		return m, nil
	}
	p, err := h.s.GetPart(id)
	if err != nil || !p.Fabricado {
		return model.Machine{}, store.ErrNotFound
	}
	return comoMaquina(p), nil
}

func comoMaquina(p model.Part) model.Machine {
	return model.Machine{
		ID: p.SKU, SKU: p.SKU, Nome: p.Nome, Apelido: "Subconjunto fabricado",
		Tecnologia: "Montado na BoxCubo", Ativa: true, Subconjunto: true,
	}
}

// Lista de subconjuntos para a barra lateral.
func (h *handler) listSubconjuntos(w http.ResponseWriter, r *http.Request) {
	out := []model.Machine{}
	for _, p := range h.s.ListSubconjuntos() {
		out = append(out, comoMaquina(p))
	}
	writeJSON(w, http.StatusOK, out)
}

// custoUnitario: preço da peça; se não tiver preço e for subconjunto,
// soma o custo da receita (ex.: CPU = Raspberry + display + Shield...).
// nil = não dá para calcular (falta preço em alguma peça sem substituto).
func (h *handler) custoUnitario(sku string, memo map[string]*float64, visitando map[string]bool) *float64 {
	if c, ok := memo[sku]; ok {
		return c
	}
	p, err := h.s.GetPart(sku)
	if err != nil {
		return nil
	}
	if p.ValorUnitario != nil || !p.Fabricado || visitando[sku] {
		memo[sku] = p.ValorUnitario
		return p.ValorUnitario
	}
	visitando[sku] = true
	itens, _ := h.s.GetBOM(sku)
	total, algum := 0.0, false
	for _, it := range itens {
		if c := h.custoUnitario(it.SKU, memo, visitando); c != nil {
			total += *c * it.Quantidade
			algum = true
		}
	}
	delete(visitando, sku)
	if !algum {
		memo[sku] = nil
		return nil
	}
	t := round2(total)
	memo[sku] = &t
	return &t
}

// usadaEm: para cada peça, em quais máquinas ela entra (direto ou dentro
// de um subconjunto). Serve para a coluna "Usada em" e o filtro
// "Só compartilhadas".
func (h *handler) usadaEm() map[string][]string {
	out := map[string]map[string]bool{}
	var desce func(dono, maquina string, visitados map[string]bool)
	desce = func(dono, maquina string, visitados map[string]bool) {
		if visitados[dono] {
			return
		}
		visitados[dono] = true
		itens, _ := h.s.GetBOM(dono)
		for _, it := range itens {
			if out[it.SKU] == nil {
				out[it.SKU] = map[string]bool{}
			}
			out[it.SKU][maquina] = true
			desce(it.SKU, maquina, visitados) // se for subconjunto, entra na receita dele
		}
	}
	for _, m := range h.s.ListMachines() {
		desce(m.ID, strings.TrimPrefix(m.Nome, "BoxCubo "), map[string]bool{})
	}
	res := map[string][]string{}
	for sku, ms := range out {
		for m := range ms {
			res[sku] = append(res[sku], m)
		}
		sort.Strings(res[sku])
	}
	return res
}

// fabricar: "Registrar fabricação" de um subconjunto.
// Body: {"quantidade": 5, "observacao": "...", "permitir_falta": false}
// Mesma regra de falta das saídas: avisa primeiro (409 + lista), e com
// permitir_falta = true tira o que tem e anota a falta.
func (h *handler) fabricar(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	p, err := h.s.GetPart(sku)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !p.Fabricado {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": sku + " não é um subconjunto fabricado"})
		return
	}
	var req struct {
		Quantidade    float64 `json:"quantidade"`
		Observacao    string  `json:"observacao"`
		PermitirFalta bool    `json:"permitir_falta"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Quantidade <= 0 || req.Quantidade != math.Trunc(req.Quantidade) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "informe quantas unidades foram fabricadas (número inteiro)"})
		return
	}
	itens, _ := h.s.GetBOM(sku)
	if len(itens) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "o subconjunto ainda não tem receita"})
		return
	}
	changes := []model.StockChange{}
	faltas := []model.Shortage{}
	for _, it := range itens {
		c, err := h.s.GetPart(it.SKU)
		if err != nil || it.Quantidade <= 0 {
			continue
		}
		need := round2(it.Quantidade * req.Quantidade)
		changes = append(changes, model.StockChange{SKU: it.SKU, Delta: -need})
		if c.Estoque < need {
			faltas = append(faltas, model.Shortage{
				SKU: c.SKU, Nome: c.Nome, Unidade: c.Unidade, Necessario: need,
				Estoque: c.Estoque, Falta: round2(need - c.Estoque), Imagem: c.Imagem,
			})
		}
	}
	if len(faltas) > 0 && !req.PermitirFalta {
		writeJSON(w, http.StatusConflict, map[string]any{
			"erro":   fmt.Sprintf("não há peças suficientes para fabricar %v: %d itens em falta", req.Quantidade, len(faltas)),
			"faltas": faltas,
		})
		return
	}
	// o subconjunto pronto entra no estoque
	changes = append(changes, model.StockChange{SKU: sku, Delta: req.Quantidade})
	mov, err := h.s.RegisterMovement(model.Movement{
		Tipo: model.MovFabricacao, Quantidade: int(req.Quantidade),
		Observacao: strings.TrimSpace(req.Observacao), MachineID: sku,
	}, changes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.comAlertaMinimo(mov))
}
