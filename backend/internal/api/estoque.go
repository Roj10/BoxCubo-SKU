// ============================================================
//  ESTOQUE: saídas de máquina, saídas de peças, entradas, estornos
// ============================================================
//  Tudo que faz o estoque subir ou descer passa por aqui e vira uma
//  MOVIMENTAÇÃO no histórico (tela "Movimentações").
//
//  Regra de falta (combinada com vocês): se for sair mais do que existe,
//  o sistema PRIMEIRO avisa (responde com a lista do que falta, e a tela
//  mostra o pop-up). Se a pessoa clicar "Continuar mesmo assim", o pedido
//  volta com permitir_falta = true: sai o que tem, a peça fica em 0 e a
//  falta fica anotada.

package api

import (
	"fmt"
	"net/http"
	"strings"

	"boxcubo-sku/internal/model"
)

// Endereços desta parte (ver o mapa completo em api.go).
func (h *handler) routesEstoque(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/machines/{id}/saida", h.saidaMaquina)
	mux.HandleFunc("POST /api/entradas", h.entrada)
	mux.HandleFunc("POST /api/saidas", h.saidaPecas)
	mux.HandleFunc("GET /api/movements", h.listMovements)
	mux.HandleFunc("GET /api/movements/{id}", h.getMovement)
	mux.HandleFunc("POST /api/movements/{id}/estorno", h.estorno)
}

// O que a tela manda no "Registrar saída" de máquina.
type saidaReq struct {
	NumerosSerie  []string `json:"numeros_serie"`
	Kits          []string `json:"kits"`
	Cliente       string   `json:"cliente"`
	Observacao    string   `json:"observacao"`
	PermitirFalta bool     `json:"permitir_falta"` // "continuar mesmo assim"
}

// saidaMaquina dá baixa na lista de peças da máquina, uma vez por número de série.
// Se faltar peça e permitir_falta=false, responde 409 com a lista do que falta;
// o frontend mostra o pop-up e, se o usuário continuar, reenvia com permitir_falta=true.
// REGISTRAR SAÍDA DE MÁQUINA. Passo a passo:
//  1. confere os números de série (obrigatórios, sem repetir, ainda não usados);
//  2. cada número de série = 1 máquina;
//  3. monta a baixa: para cada peça da lista, quantidade x nº de máquinas
//     (itens de kit só se o kit foi marcado);
//  4. se faltar peça e a pessoa ainda não confirmou -> devolve o aviso;
//  5. se não, registra a movimentação e devolve também quais peças
//     ficaram abaixo do estoque mínimo.
func (h *handler) saidaMaquina(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	machine, err := h.s.GetMachine(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	var req saidaReq
	if !readJSON(w, r, &req) {
		return
	}

	// números de série: obrigatórios, sem repetição, ainda não usados
	series := []string{}
	vistos := map[string]bool{}
	for _, ns := range req.NumerosSerie {
		ns = strings.ToUpper(strings.TrimSpace(ns))
		if ns == "" {
			continue
		}
		if vistos[ns] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "número de série repetido: " + ns})
			return
		}
		vistos[ns] = true
		if mov, usado := h.s.SerialEmUso(ns); usado {
			writeJSON(w, http.StatusConflict, map[string]string{
				"erro": fmt.Sprintf("o número de série %s já saiu em %s", ns, mov),
			})
			return
		}
		series = append(series, ns)
	}
	if len(series) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "informe pelo menos um número de série"})
		return
	}
	qtd := float64(len(series))

	kits := map[string]bool{}
	for _, k := range req.Kits {
		kits[k] = true
	}
	items, err := h.s.GetBOM(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	changes := []model.StockChange{}
	faltas := []model.Shortage{}
	for _, it := range items {
		if (it.Kit != "" && !kits[it.Kit]) || it.Quantidade <= 0 {
			continue
		}
		p, err := h.s.GetPart(it.SKU)
		if err != nil {
			continue
		}
		need := round2(it.Quantidade * qtd)
		changes = append(changes, model.StockChange{SKU: it.SKU, Delta: -need})
		if p.Estoque < need {
			faltas = append(faltas, model.Shortage{
				SKU: p.SKU, Nome: p.Nome, Unidade: p.Unidade,
				Necessario: need, Estoque: p.Estoque, Falta: round2(need - p.Estoque), Imagem: p.Imagem,
			})
		}
	}

	// Falta peça e ainda não houve "Continuar mesmo assim": só avisa,
	// NADA sai do estoque nesta hora.
	if len(faltas) > 0 && !req.PermitirFalta {
		writeJSON(w, http.StatusConflict, map[string]any{
			"erro":   fmt.Sprintf("não há peças suficientes para %d máquina(s): %d itens em falta", len(series), len(faltas)),
			"faltas": faltas,
		})
		return
	}

	kitList := []string{}
	for _, k := range req.Kits {
		if strings.TrimSpace(k) != "" {
			kitList = append(kitList, k)
		}
	}
	mov, err := h.s.RegisterMovement(model.Movement{
		Tipo: model.MovSaidaMaquina, MachineID: machine.ID, Quantidade: len(series),
		NumerosSerie: series, Kits: kitList, Cliente: strings.TrimSpace(req.Cliente),
		Observacao: strings.TrimSpace(req.Observacao),
	}, changes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.comAlertaMinimo(mov))
}

// O que a tela manda no "Registrar entrada": lista de {peça, quantidade}.
type entradaReq struct {
	Itens []struct {
		SKU        string  `json:"sku"`
		Quantidade float64 `json:"quantidade"`
	} `json:"itens"`
	Observacao string `json:"observacao"`
}

// entrada soma ao estoque o material que chegou.
// REGISTRAR ENTRADA: material que chegou. Soma cada quantidade ao estoque.
func (h *handler) entrada(w http.ResponseWriter, r *http.Request) {
	var req entradaReq
	if !readJSON(w, r, &req) {
		return
	}
	changes := []model.StockChange{}
	for _, it := range req.Itens {
		if it.Quantidade < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "quantidade negativa em " + it.SKU})
			return
		}
		if it.Quantidade > 0 {
			changes = append(changes, model.StockChange{SKU: it.SKU, Delta: it.Quantidade})
		}
	}
	if len(changes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "nenhuma peça com quantidade para dar entrada"})
		return
	}
	mov, err := h.s.RegisterMovement(model.Movement{
		Tipo: model.MovEntrada, Observacao: strings.TrimSpace(req.Observacao),
	}, changes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mov)
}

// O que a tela manda no "Registrar saída de peças" (avulsas).
type saidaPecasReq struct {
	Itens []struct {
		SKU        string  `json:"sku"`
		Quantidade float64 `json:"quantidade"`
	} `json:"itens"`
	Cliente       string `json:"cliente"`
	NumeroSerie   string `json:"numero_serie"` // máquina do cliente (opcional)
	Observacao    string `json:"observacao"`
	PermitirFalta bool   `json:"permitir_falta"`
}

// saidaPecas tira do estoque peças avulsas (manutenção, reposição para cliente).
// Mesma regra da saída de máquina: se faltar, 409 com a lista; com
// permitir_falta=true tira o que existe e deixa a peça em 0.
// SAÍDA DE PEÇAS AVULSAS: ex. peças enviadas para manutenção no cliente.
// Mesma regra de falta da saída de máquina. Se a mesma peça aparecer em
// duas linhas, as quantidades são somadas.
func (h *handler) saidaPecas(w http.ResponseWriter, r *http.Request) {
	var req saidaPecasReq
	if !readJSON(w, r, &req) {
		return
	}
	somado := map[string]float64{} // mesma peça em duas linhas vira uma
	ordem := []string{}
	for _, it := range req.Itens {
		if it.Quantidade < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "quantidade negativa em " + it.SKU})
			return
		}
		if it.Quantidade == 0 {
			continue
		}
		if _, ok := somado[it.SKU]; !ok {
			ordem = append(ordem, it.SKU)
		}
		somado[it.SKU] += it.Quantidade
	}
	if len(ordem) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "nenhuma peça com quantidade para sair"})
		return
	}
	changes := []model.StockChange{}
	faltas := []model.Shortage{}
	for _, sku := range ordem {
		p, err := h.s.GetPart(sku)
		if err != nil {
			writeErr(w, err)
			return
		}
		q := round2(somado[sku])
		changes = append(changes, model.StockChange{SKU: sku, Delta: -q})
		if p.Estoque < q {
			faltas = append(faltas, model.Shortage{
				SKU: p.SKU, Nome: p.Nome, Unidade: p.Unidade, Necessario: q,
				Estoque: p.Estoque, Falta: round2(q - p.Estoque), Imagem: p.Imagem,
			})
		}
	}
	if len(faltas) > 0 && !req.PermitirFalta {
		writeJSON(w, http.StatusConflict, map[string]any{
			"erro":   fmt.Sprintf("não há estoque suficiente: %d peça(s) em falta", len(faltas)),
			"faltas": faltas,
		})
		return
	}
	mov := model.Movement{
		Tipo: model.MovSaidaPecas, Cliente: strings.TrimSpace(req.Cliente),
		Observacao: strings.TrimSpace(req.Observacao),
	}
	if ns := strings.ToUpper(strings.TrimSpace(req.NumeroSerie)); ns != "" {
		mov.NumerosSerie = []string{ns}
	}
	out, err := h.s.RegisterMovement(mov, changes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.comAlertaMinimo(out))
}

// listMovements aceita ?machine=v2&tipo=saida_maquina&q=SERIE
// HISTÓRICO (tela "Movimentações"), com filtros por tipo e busca por
// número de série, cliente, código MOV-...
func (h *handler) listMovements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	machine, tipo := q.Get("machine"), q.Get("tipo")
	busca := strings.ToUpper(strings.TrimSpace(q.Get("q")))
	out := []model.Movement{}
	for _, m := range h.s.ListMovements() {
		if machine != "" && m.MachineID != machine {
			continue
		}
		if tipo != "" && m.Tipo != tipo {
			continue
		}
		if busca != "" {
			txt := strings.ToUpper(m.ID + " " + m.Cliente + " " + m.Observacao + " " + strings.Join(m.NumerosSerie, " "))
			if !strings.Contains(txt, busca) {
				continue
			}
		}
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) getMovement(w http.ResponseWriter, r *http.Request) {
	m, err := h.s.GetMovement(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Botão "Estornar": desfaz uma movimentação lançada errada.
func (h *handler) estorno(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Observacao string `json:"observacao"`
	}
	if r.ContentLength > 0 && !readJSON(w, r, &req) {
		return
	}
	m, err := h.s.ReverseMovement(r.PathValue("id"), strings.TrimSpace(req.Observacao))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}
