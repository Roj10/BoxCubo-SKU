// TESTES DO SUPABASE: usa um "Supabase de mentira" (um servidorzinho que
// imita a API REST dele) para conferir que os dados vão e voltam do banco
// sem perder nada.
package store

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"boxcubo-sku/internal/model"
)

// supabaseFalso guarda as tabelas na memória e responde como o PostgREST.
type supabaseFalso struct {
	mu      sync.Mutex
	tabs    map[string][]map[string]any
	pedidos int
}

func novoSupabaseFalso() *supabaseFalso {
	f := &supabaseFalso{tabs: map[string][]map[string]any{}}
	for _, tb := range tabelas {
		f.tabs[tb.nome] = []map[string]any{}
	}
	return f
}

func chaveDaTabela(nome string) []string {
	for _, tb := range tabelas {
		if tb.nome == nome {
			return tb.chave
		}
	}
	return nil
}

// confere um filtro do PostgREST: "eq.x" ou "in.("a","b")"
func casa(filtro string, v any) bool {
	val := valorTexto(v)
	if strings.HasPrefix(filtro, "eq.") {
		return val == filtro[3:]
	}
	if strings.HasPrefix(filtro, "in.(") {
		lista := strings.TrimSuffix(strings.TrimPrefix(filtro, "in.("), ")")
		for _, item := range strings.Split(lista, `","`) {
			item = strings.Trim(item, `"`)
			item = strings.ReplaceAll(item, `\"`, `"`)
			item = strings.ReplaceAll(item, `\\`, `\`)
			if item == val {
				return true
			}
		}
	}
	return false
}

func (f *supabaseFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pedidos++
	if r.Header.Get("apikey") != "sb_secret_teste" {
		http.Error(w, `{"message":"Invalid API key"}`, 401)
		return
	}
	nome := strings.TrimPrefix(r.URL.Path, "/rest/v1/")
	linhas, ok := f.tabs[nome]
	if !ok {
		http.Error(w, `{"code":"PGRST205","message":"Could not find the table"}`, 404)
		return
	}
	q := r.URL.Query()
	switch r.Method {
	case "GET":
		cols := strings.Split(q.Get("order"), ",")
		cp := append([]map[string]any(nil), linhas...)
		sort.SliceStable(cp, func(i, j int) bool {
			for _, c := range cols {
				a, b := cp[i][c], cp[j][c]
				if fa, ok := a.(float64); ok {
					if fa != b.(float64) {
						return fa < b.(float64)
					}
					continue
				}
				if valorTexto(a) != valorTexto(b) {
					return valorTexto(a) < valorTexto(b)
				}
			}
			return false
		})
		off, _ := strconv.Atoi(q.Get("offset"))
		lim, _ := strconv.Atoi(q.Get("limit"))
		if off > len(cp) {
			off = len(cp)
		}
		fim := min(off+lim, len(cp))
		json.NewEncoder(w).Encode(cp[off:fim])
	case "POST":
		chave := strings.Split(q.Get("on_conflict"), ",")
		var novas []map[string]any
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &novas); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		for _, n := range novas {
			trocou := false
			for i, l := range linhas {
				igual := true
				for _, c := range chave {
					if valorTexto(l[c]) != valorTexto(n[c]) {
						igual = false
					}
				}
				if igual {
					linhas[i], trocou = n, true
					break
				}
			}
			if !trocou {
				linhas = append(linhas, n)
			}
		}
		f.tabs[nome] = linhas
		w.WriteHeader(201)
	case "DELETE":
		ficam := []map[string]any{}
		for _, l := range linhas {
			sai := len(q) > 0
			for c, v := range q {
				if !casa(v[0], l[c]) {
					sai = false
				}
			}
			if !sai {
				ficam = append(ficam, l)
			}
		}
		f.tabs[nome] = ficam
		w.WriteHeader(204)
	}
}

// fotoDoServidor: o conteúdo do "banco" no mesmo formato da foto.
func (f *supabaseFalso) foto(t *testing.T) foto {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := foto{}
	for _, tb := range tabelas {
		out[tb.nome] = map[string]string{}
		for _, l := range f.tabs[tb.nome] {
			raw, _ := json.Marshal(l)
			out[tb.nome][chaveDe(tb, l)] = string(raw)
		}
	}
	return out
}

// mesma informação, sem ligar para a ordem das colunas no JSON
func normalizar(f foto) foto {
	out := foto{}
	for t, linhas := range f {
		out[t] = map[string]string{}
		for k, raw := range linhas {
			var m map[string]any
			_ = json.Unmarshal([]byte(raw), &m)
			b, _ := json.Marshal(m)
			out[t][k] = string(b)
		}
	}
	return out
}

func compararFotos(t *testing.T, onde string, a, b foto) {
	t.Helper()
	a, b = normalizar(a), normalizar(b)
	for _, tb := range tabelas {
		if len(a[tb.nome]) != len(b[tb.nome]) {
			t.Fatalf("%s: tabela %s com %d x %d linhas", onde, tb.nome, len(a[tb.nome]), len(b[tb.nome]))
		}
		for k, v := range a[tb.nome] {
			if b[tb.nome][k] != v {
				t.Fatalf("%s: tabela %s, linha %q diferente:\n %s\n %s", onde, tb.nome, k, v, b[tb.nome][k])
			}
		}
	}
}

func TestSupabaseIdaEVolta(t *testing.T) {
	falso := novoSupabaseFalso()
	srv := httptest.NewServer(falso)
	defer srv.Close()
	dir := t.TempDir()

	// 1) banco vazio, sem arquivo local: grava as peças iniciais da V2
	s, origem, err := NewSupabaseStore(srv.URL, "sb_secret_teste", filepath.Join(dir, "nao-existe.json"))
	if err != nil || !strings.Contains(origem, "peças iniciais") {
		t.Fatalf("abrir: %v %q", err, origem)
	}
	if n := len(falso.tabs["pecas"]); n != len(s.ListParts("")) || n < 100 {
		t.Fatalf("peças no banco: %d", n)
	}

	// 2) as atualizações de dados (migrações) mexem em quase tudo
	if _, err := s.Migrar(dir); err != nil {
		t.Fatal(err)
	}
	// 3) operações do dia a dia
	if _, err := s.SetMinimo("v2", 10, false); err != nil {
		t.Fatal(err)
	}
	preco := 1.5
	if _, err := s.CreatePart("ELE-RES", "47K", model.Part{Nome: "Resistor 47k", Unidade: "und", Tipo: "peca",
		ValorUnitario: &preco, Fornecedores: []model.Fornecimento{{Nome: "Fornell", Codigo: "R47K, (5%)"}, {Nome: "Mercado Livre"}}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertBOMItem("v2", model.BOMItem{SKU: "ELE-RES-47K", Quantidade: 2, OndeUsado: []string{"PR"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenameSKU("ELE-RES-47K", "ELE-RES-47KB"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.SubstituirFornecedor("Mercado Livre", ""); err != nil {
		t.Fatal(err)
	}
	mov, err := s.RegisterMovement(model.Movement{Tipo: model.MovEntrada, Observacao: `nota "123"`},
		[]model.StockChange{{SKU: "ELE-RES-47KB", Delta: 30}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReverseMovement(mov.ID, "errei"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteBOMItem("v2", "ELE-RES-47KB"); err != nil {
		t.Fatal(err)
	}

	// o banco tem que estar igual ao que está na memória
	naMemoria, _ := tirarFoto(&s.d)
	compararFotos(t, "banco x memória", naMemoria, falso.foto(t))

	// 4) religar o backend: lê tudo do banco e fica igual
	pedidosAntes := falso.pedidos
	s2, origem, err := NewSupabaseStore(srv.URL, "sb_secret_teste", "")
	if err != nil || origem != "dados lidos do Supabase" {
		t.Fatalf("reabrir: %v %q", err, origem)
	}
	relido, _ := tirarFoto(&s2.d)
	compararFotos(t, "relido x memória", naMemoria, relido)
	p, _ := s2.GetPart("ELE-RES-47KB")
	if p.ValorUnitario == nil || *p.ValorUnitario != 1.5 || len(p.Fornecedores) != 1 || p.Fornecedores[0].Codigo != "R47K, (5%)" {
		t.Fatalf("peça relida: %+v", p)
	}
	if m, _ := s2.GetMachine("v2"); m.MinimoMaquinas != 10 {
		t.Fatalf("mínimo relido: %+v", m)
	}
	// nada mudou => reabrir não grava nada (só leituras, uma por tabela)
	if falso.pedidos-pedidosAntes != len(tabelas) {
		t.Fatalf("reabrir fez %d pedidos", falso.pedidos-pedidosAntes)
	}
	// 5) uma mudança pequena manda só o necessário
	antes := falso.pedidos
	if _, err := s2.SetMinimo("eye", 30, false); err != nil {
		t.Fatal(err)
	}
	if falso.pedidos-antes != 1 {
		t.Fatalf("mudar um mínimo fez %d pedidos (esperado 1)", falso.pedidos-antes)
	}
}

func TestSupabaseCopiaArquivoLocal(t *testing.T) {
	falso := novoSupabaseFalso()
	srv := httptest.NewServer(falso)
	defer srv.Close()
	// arquivo local com dados (como o boxcubo.json do usuário)
	dir := t.TempDir()
	arq := filepath.Join(dir, "boxcubo.json")
	fs, err := NewFileStore(arq)
	if err != nil {
		t.Fatal(err)
	}
	fs.SetMinimo("v2", 7, true)
	_, origem, err := NewSupabaseStore(srv.URL, "sb_secret_teste", arq)
	if err != nil || !strings.Contains(origem, "copiados de") {
		t.Fatalf("%v %q", err, origem)
	}
	achou := false
	for _, m := range falso.tabs["maquinas"] {
		if m["id"] == "v2" && m["minimo_maquinas"] == float64(7) {
			achou = true
		}
	}
	if !achou {
		t.Fatal("o mínimo do arquivo local deveria ter ido para o banco")
	}
	// o arquivo local continua lá (cópia de segurança)
	if _, err := os.Stat(arq); err != nil {
		t.Fatal(err)
	}
}

func TestSupabaseErros(t *testing.T) {
	// tabela faltando
	vazio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"PGRST205","message":"Could not find the table 'public.maquinas'"}`, 404)
	}))
	defer vazio.Close()
	if _, _, err := NewSupabaseStore(vazio.URL, "x", ""); !errors.Is(err, ErrTabelasFaltando) {
		t.Fatalf("esperava aviso de tabelas faltando: %v", err)
	}
	// chave errada
	falso := novoSupabaseFalso()
	srv := httptest.NewServer(falso)
	defer srv.Close()
	if _, _, err := NewSupabaseStore(srv.URL, "sb_publishable_x", ""); err == nil || !strings.Contains(err.Error(), "SECRETA") {
		t.Fatalf("esperava erro de chave: %v", err)
	}
}
