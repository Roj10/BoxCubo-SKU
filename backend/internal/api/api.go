// ============================================================
//  API: o "balcão" entre a tela e a despensa
// ============================================================
//  A tela (frontend) nunca mexe nos dados direto. Ela faz PEDIDOS para
//  endereços como /api/machines/v2/bom ("me dá a lista de peças da V2").
//  Este arquivo recebe cada pedido, confere se está certo, pede os dados
//  para o store, faz as contas e devolve a resposta.
//
//  Os pedidos têm um "verbo":
//    GET    = consultar          (ex.: ver a lista)
//    POST   = criar              (ex.: peça nova, saída)
//    PUT    = alterar            (ex.: mudar preço)
//    DELETE = apagar / tirar     (ex.: tirar peça da máquina)
//  As respostas e pedidos vão no formato JSON (texto organizado).

// Package api expõe o sistema de SKU como uma API REST em JSON.
package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"boxcubo-sku/internal/model"
	"boxcubo-sku/internal/store"
)

// New monta as rotas da API. imgDir é a pasta onde ficam as fotos das peças.
// MAPA DE ENDEREÇOS: liga cada endereço a quem responde por ele.
// Ex.: "GET /api/machines/{id}/bom" -> getBOM (lista de peças da máquina).
func New(s store.Store, imgDir string) http.Handler {
	h := &handler{s: s, imgDir: imgDir}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/machines", h.listMachines)
	mux.HandleFunc("GET /api/machines/{id}", h.getMachine)
	mux.HandleFunc("GET /api/machines/{id}/bom", h.getBOM)
	mux.HandleFunc("PUT /api/machines/{id}/bom/{sku}", h.upsertBOM)
	mux.HandleFunc("DELETE /api/machines/{id}/bom/{sku}", h.deleteBOM)
	mux.HandleFunc("GET /api/machines/{id}/plan", h.plan)
	mux.HandleFunc("GET /api/kits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.ListKits())
	})
	mux.HandleFunc("GET /api/legenda", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.Legenda())
	})
	mux.HandleFunc("GET /api/fornecedores", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.ListFornecedores())
	})
	mux.HandleFunc("GET /api/classes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.ListClasses())
	})
	mux.HandleFunc("GET /api/parts", h.listParts)
	mux.HandleFunc("POST /api/parts", h.createPart)
	mux.HandleFunc("GET /api/parts/{sku}", h.getPart)
	mux.HandleFunc("PUT /api/parts/{sku}", h.updatePart)
	mux.HandleFunc("DELETE /api/parts/{sku}", h.deletePart)
	mux.HandleFunc("PUT /api/parts/{sku}/sku", h.renameSKU)
	h.routesEstoque(mux)
	h.routesImagens(mux)
	h.routesMinimo(mux)
	h.routesSubconjuntos(mux)
	h.routesFornecedores(mux)
	return cors(mux)
}

// O "atendente" do balcão: tem acesso à despensa (s) e à pasta de fotos.
type handler struct {
	s      store.Store
	imgDir string
}

func (h *handler) listMachines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.s.ListMachines())
}

func (h *handler) getMachine(w http.ResponseWriter, r *http.Request) {
	m, err := h.s.GetMachine(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// O que a tela recebe ao abrir uma máquina: a lista + custos + contagens.
type bomResponse struct {
	// Peca: preenchido quando a lista é de um subconjunto (para mostrar o estoque dele).
	Peca          *model.Part        `json:"peca,omitempty"`
	Machine       model.Machine      `json:"machine"`
	Linhas        []model.BOMLine    `json:"linhas"`
	AbaixoMinimo  int                `json:"abaixo_minimo"`
	CustoPadrao   float64            `json:"custo_padrao"` // sem kits
	CustoPorKit   map[string]float64 `json:"custo_por_kit"`
	ItensSemPreco int                `json:"itens_sem_preco"`
}

// LISTA DE PEÇAS DA MÁQUINA (aba "Lista de peças").
// Para cada item: junta os dados da peça, calcula o custo por máquina
// (quantidade x preço) e soma os totais: custo padrão (sem kits) e custo de
// cada kit. Também conta peças sem preço e abaixo do estoque mínimo.
func (h *handler) getBOM(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, err := h.dono(id) // máquina ou subconjunto (ex.: CPU)
	if err != nil {
		writeErr(w, err)
		return
	}
	items, err := h.s.GetBOM(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	resp := bomResponse{Machine: m, Linhas: []model.BOMLine{}, CustoPorKit: map[string]float64{}}
	if m.Subconjunto {
		if p, err := h.s.GetPart(id); err == nil {
			resp.Peca = &p
		}
	}
	min, _ := h.minimos()
	usada := h.usadaEm()
	memo := map[string]*float64{}
	for _, it := range items {
		p, err := h.s.GetPart(it.SKU)
		if err != nil {
			continue
		}
		line := model.BOMLine{BOMItem: it, Peca: p, Minimo: min[it.SKU], UsadaEm: usada[it.SKU]}
		if line.UsadaEm == nil {
			line.UsadaEm = []string{}
		}
		line.CustoUnitario = h.custoUnitario(it.SKU, memo, map[string]bool{})
		if line.CustoUnitario != nil {
			c := round2(*line.CustoUnitario * it.Quantidade)
			line.CustoPorMaquina = &c
			if it.Kit == "" {
				resp.CustoPadrao += c
			} else {
				resp.CustoPorKit[it.Kit] += c
			}
		} else {
			resp.ItensSemPreco++
		}
		resp.Linhas = append(resp.Linhas, line)
	}
	// peças abaixo do mínimo nesta máquina, incluindo as que vão dentro dos
	// subconjuntos dela (ex.: resistores do Shield, dentro da CPU)
	vistos := map[string]bool{}
	var conta func(dono string)
	conta = func(dono string) {
		itens, _ := h.s.GetBOM(dono)
		for _, it := range itens {
			if vistos[it.SKU] {
				continue
			}
			vistos[it.SKU] = true
			if p, err := h.s.GetPart(it.SKU); err == nil {
				if min[it.SKU] > 0 && p.Estoque < min[it.SKU] {
					resp.AbaixoMinimo++
				}
				if p.Fabricado {
					conta(it.SKU)
				}
			}
		}
	}
	conta(id)
	resp.CustoPadrao = round2(resp.CustoPadrao)
	for k, v := range resp.CustoPorKit {
		resp.CustoPorKit[k] = round2(v)
	}
	writeJSON(w, http.StatusOK, resp)
}

// Coloca/atualiza uma peça na lista da máquina.
func (h *handler) upsertBOM(w http.ResponseWriter, r *http.Request) {
	var it model.BOMItem
	if !readJSON(w, r, &it) {
		return
	}
	it.SKU = r.PathValue("sku")
	if err := h.s.UpsertBOMItem(r.PathValue("id"), it); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// Tira uma peça da lista da máquina.
func (h *handler) deleteBOM(w http.ResponseWriter, r *http.Request) {
	if err := h.s.DeleteBOMItem(r.PathValue("id"), r.PathValue("sku")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// plan calcula a compra para um lote: ?quantidade=43&margem=10&kits=ADDC
// PLANEJAR LOTE. Ex.: 43 máquinas com 10% de margem.
// Para cada peça da lista:
//
//	necessário = quantidade por máquina x número de máquinas
//	com margem = necessário + 10% (peças inteiras são arredondadas para cima)
//	comprar    = com margem - estoque (nunca negativo)
//	custo      = comprar x preço
//
// Itens de kit só entram se o kit foi marcado.
func (h *handler) plan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	items, err := h.s.GetBOM(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	q := r.URL.Query()
	qtd, err := strconv.Atoi(q.Get("quantidade"))
	if err != nil || qtd < 1 {
		qtd = 1
	}
	margem, err := strconv.ParseFloat(q.Get("margem"), 64)
	if err != nil || margem < 0 {
		margem = 0
	}
	kits := map[string]bool{}
	kitList := []string{}
	for _, k := range strings.Split(q.Get("kits"), ",") {
		if k = strings.TrimSpace(k); k != "" {
			kits[k] = true
			kitList = append(kitList, k)
		}
	}

	plan := model.Plan{MachineID: id, Quantidade: qtd, Margem: margem, Kits: kitList, Linhas: []model.PlanLine{}}
	arred := func(p model.Part, v float64) float64 {
		if p.Tipo == "peca" || p.Tipo == "servico" {
			return math.Ceil(v - 1e-9) // peças inteiras
		}
		return round2(v)
	}

	// 1) necessidade das peças da lista (com margem)
	linhas := map[string]*model.PlanLine{}
	ordem := []string{}
	partes := map[string]model.Part{}
	pega := func(sku string) (*model.PlanLine, bool) {
		if l, ok := linhas[sku]; ok {
			return l, true
		}
		p, err := h.s.GetPart(sku)
		if err != nil {
			return nil, false
		}
		partes[sku] = p
		l := &model.PlanLine{
			SKU: p.SKU, Nome: p.Nome, Unidade: p.Unidade, Estoque: p.Estoque,
			ValorUnit: p.ValorUnitario, Fornecedores: p.Fornecedores, Imagem: p.Imagem,
			Fabricar: p.Fabricado,
		}
		linhas[sku] = l
		ordem = append(ordem, sku)
		return l, true
	}
	memo := map[string]*float64{}
	for _, it := range items {
		if it.Kit != "" && !kits[it.Kit] {
			continue
		}
		l, ok := pega(it.SKU)
		if !ok {
			continue
		}
		nec := it.Quantidade * float64(qtd)
		l.Kit = it.Kit
		l.PorMaquina += it.Quantidade
		l.Necessario += nec
		l.ComMargem += arred(partes[it.SKU], nec*(1+margem/100))
		if c := h.custoUnitario(it.SKU, memo, map[string]bool{}); c != nil {
			plan.CustoLote += nec * *c
		}
	}

	// 2) SUBCONJUNTOS: se faltar CPU, as CPUs que faltam precisam ser
	// fabricadas -> as peças da receita delas entram no plano (e assim por
	// diante: o Shield dentro da CPU). Repete até nada mudar.
	feito := map[string]float64{} // quanto de cada subconjunto já foi desdobrado
	for mudou := true; mudou; {
		mudou = false
		for _, sku := range append([]string(nil), ordem...) {
			l := linhas[sku]
			if !l.Fabricar {
				continue
			}
			fazer := math.Max(0, l.ComMargem-l.Estoque)
			if fazer <= feito[sku] {
				continue
			}
			extra := fazer - feito[sku]
			feito[sku] = fazer
			receita, _ := h.s.GetBOM(sku)
			for _, it := range receita {
				c, ok := pega(it.SKU)
				if !ok {
					continue
				}
				q := extra * it.Quantidade
				c.Necessario += q
				c.ComMargem += arred(partes[it.SKU], q)
				if !contem(c.Via, l.Nome) {
					c.Via = append(c.Via, l.Nome)
				}
			}
			mudou = true
		}
	}

	// 3) comprar (ou fabricar) = necessidade com margem - estoque
	for _, sku := range ordem {
		l := linhas[sku]
		l.Necessario = round2(l.Necessario)
		l.ComMargem = round2(l.ComMargem)
		l.Comprar = math.Max(0, round2(l.ComMargem-l.Estoque))
		if l.Fabricar {
			// subconjunto: não se compra, fabrica (o custo está nas peças dele)
		} else if l.ValorUnit != nil {
			c := round2(l.Comprar * *l.ValorUnit)
			l.CustoCompra = &c
			plan.CustoCompra += c
		} else {
			plan.ItensSemPreco++
		}
		if l.Comprar > 0 {
			plan.ItensFaltando++
		}
		plan.Linhas = append(plan.Linhas, *l)
	}
	sort.SliceStable(plan.Linhas, func(i, j int) bool { return plan.Linhas[i].SKU < plan.Linhas[j].SKU })
	plan.CustoCompra = round2(plan.CustoCompra)
	plan.CustoLote = round2(plan.CustoLote)
	writeJSON(w, http.StatusOK, plan)
}

// Lista/busca peças (usado na entrada e saída de peças avulsas).
func (h *handler) listParts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.s.ListParts(r.URL.Query().Get("q")))
}

func (h *handler) getPart(w http.ResponseWriter, r *http.Request) {
	p, err := h.s.GetPart(r.PathValue("sku"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// O que a tela manda para criar uma peça: os dados da peça + a classe
// (prefixo) + o final do SKU + (opcional) já colocar na lista da máquina.
type createPartReq struct {
	model.Part
	Prefixo string `json:"prefixo"`
	Sufixo  string `json:"sufixo"` // final do SKU (opcional)
	// Se informado, a peça já entra na lista dessa máquina.
	MachineID  string   `json:"machine_id"`
	Quantidade float64  `json:"quantidade"`
	OndeUsado  []string `json:"onde_usado"`
	Kit        string   `json:"kit"`
}

// "+ Nova peça": cria a peça e, se pedido, já coloca na lista da máquina.
func (h *handler) createPart(w http.ResponseWriter, r *http.Request) {
	var req createPartReq
	if !readJSON(w, r, &req) {
		return
	}
	p, err := h.s.CreatePart(req.Prefixo, req.Sufixo, req.Part)
	if err != nil {
		writeErr(w, err)
		return
	}
	if req.MachineID != "" {
		err := h.s.UpsertBOMItem(req.MachineID, model.BOMItem{
			SKU: p.SKU, Quantidade: req.Quantidade, OndeUsado: req.OndeUsado, Kit: req.Kit,
		})
		if err != nil {
			writeErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, p)
}

// Salva alterações de uma peça (botão "Salvar" na janela da peça).
func (h *handler) updatePart(w http.ResponseWriter, r *http.Request) {
	var p model.Part
	if !readJSON(w, r, &p) {
		return
	}
	out, err := h.s.UpdatePart(r.PathValue("sku"), p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// renameSKU troca o final do SKU: {"sufixo": "1K2"} -> BC-ELE-RES-1K2.
// Troca o final do SKU (campo "Final do SKU" na janela da peça).
// Ex.: ELE-RES-10K -> ELE-RES-10K0, levando junto lista, histórico e foto.
func (h *handler) renameSKU(w http.ResponseWriter, r *http.Request) {
	antigo := r.PathValue("sku")
	var req struct {
		Sufixo string `json:"sufixo"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	sufixo, err := store.NormalizarSufixo(req.Sufixo)
	if err != nil {
		writeErr(w, err)
		return
	}
	i := strings.LastIndex(antigo, "-")
	if i < 0 {
		writeErr(w, store.ErrInvalid)
		return
	}
	novo := antigo[:i] + "-" + sufixo
	p, err := h.s.RenameSKU(antigo, novo)
	if err != nil {
		writeErr(w, err)
		return
	}
	store.RenomearArquivosImagem(h.imgDir, antigo, novo)
	writeJSON(w, http.StatusOK, p)
}

// Apaga uma peça do cadastro (só se não estiver em nenhuma máquina).
func (h *handler) deletePart(w http.ResponseWriter, r *http.Request) {
	if err := h.s.DeletePart(r.PathValue("sku")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- utilitários ----

// contem diz se o texto está na lista.
func contem(lista []string, txt string) bool {
	for _, x := range lista {
		if x == txt {
			return true
		}
	}
	return false
}

// ------------------------- FERRAMENTAS ----------------------------
// Arredonda para 2 casas (centavos): 8.698376 -> 8.70.
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// Lê o pedido que veio da tela. Se vier quebrado, responde "JSON inválido".
// Também limita o tamanho do pedido (1 MB) por segurança.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "JSON inválido: " + err.Error()})
		return false
	}
	return true
}

// Manda a resposta para a tela. "status" é o código do resultado:
// 200 = ok, 201 = criado, 400 = pedido errado, 404 = não achou, 409 = conflito.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Transforma um erro interno numa resposta clara para a tela, com o código
// certo (ex.: peça não encontrada -> 404).
func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, store.ErrInvalid):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"erro": err.Error()})
}

// cors libera o frontend de desenvolvimento (Vite em localhost:5173).
// Permissão de segurança do navegador: libera a tela que roda no próprio
// computador (localhost) para conversar com este servidor.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
