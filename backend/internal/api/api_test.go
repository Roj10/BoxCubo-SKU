// TESTES AUTOMÁTICOS (lista de peças, cadastro e planejamento).
// Rode com:  go test ./...   (na pasta backend)
// Cada teste cria um arquivo de dados temporário, faz pedidos como a tela
// faria e confere se as respostas e as contas estão certas. Se algo quebrar
// numa mudança futura, o teste avisa antes de chegar em vocês.

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"boxcubo-sku/internal/model"
	"boxcubo-sku/internal/store"
)

func setup(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	s, err := store.NewFileStore(filepath.Join(dir, "db.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Migrar(filepath.Join(dir, "imagens")); err != nil {
		t.Fatal(err)
	}
	return New(s, filepath.Join(dir, "imagens"))
}

func do(t *testing.T, h http.Handler, method, url string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, url, &buf))
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: %v (%s)", method, url, err, rec.Body.String())
		}
	}
	return rec.Code
}

func TestBOMV2(t *testing.T) {
	h := setup(t)
	var bom bomResponse
	if c := do(t, h, "GET", "/api/machines/v2/bom", nil, &bom); c != 200 {
		t.Fatalf("status %d", c)
	}
	if len(bom.Linhas) != 123 {
		t.Fatalf("esperava 123 linhas, veio %d", len(bom.Linhas))
	}
	if bom.CustoPadrao <= 3850 || bom.CustoPorKit["ADDC"] <= 0 {
		t.Fatalf("custos estranhos: %+v %+v", bom.CustoPadrao, bom.CustoPorKit)
	}
	t.Logf("custo padrão V2: R$ %.2f | ADDC: R$ %.2f | sem preço: %d",
		bom.CustoPadrao, bom.CustoPorKit["ADDC"], bom.ItensSemPreco)
}

func TestCreateUpdateDelete(t *testing.T) {
	h := setup(t)
	v := 0.05
	var p model.Part
	code := do(t, h, "POST", "/api/parts", map[string]any{
		"prefixo": "ELE-RES", "nome": "47 Ω", "valor_unitario": v,
		"machine_id": "v2", "quantidade": 2, "onde_usado": []string{"PR"},
	}, &p)
	// sem final escolhido: número automático
	if code != 201 || p.SKU != "ELE-RES-001" {
		t.Fatalf("create: %d %+v", code, p)
	}
	p.Nome = "47 Ω 1/4W"
	var up model.Part
	if c := do(t, h, "PUT", "/api/parts/"+p.SKU, p, &up); c != 200 || up.Nome != "47 Ω 1/4W" {
		t.Fatalf("update: %d %+v", c, up)
	}
	if c := do(t, h, "DELETE", "/api/parts/"+p.SKU, nil, nil); c != 409 {
		t.Fatalf("deveria bloquear exclusão de peça em uso, veio %d", c)
	}
	if c := do(t, h, "DELETE", "/api/machines/v2/bom/"+p.SKU, nil, nil); c != 204 {
		t.Fatalf("remover da BOM: %d", c)
	}
	if c := do(t, h, "DELETE", "/api/parts/"+p.SKU, nil, nil); c != 204 {
		t.Fatalf("excluir: %d", c)
	}
}

func TestPlan(t *testing.T) {
	h := setup(t)
	var sem, com model.Plan
	do(t, h, "GET", "/api/machines/v2/plan?quantidade=43&margem=10", nil, &sem)
	do(t, h, "GET", "/api/machines/v2/plan?quantidade=43&margem=10&kits=ADDC", nil, &com)
	if len(com.Linhas) <= len(sem.Linhas) || com.CustoLote <= sem.CustoLote {
		t.Fatalf("kit ADDC não entrou no plano")
	}
	for _, l := range sem.Linhas {
		if l.Nome == "Balança" && (l.Necessario != 43 || l.ComMargem != 48 || l.Comprar != 21) {
			t.Fatalf("balança: %+v", l)
		}
	}
	t.Logf("lote 43 (+10%%): custo lote R$ %.2f, comprar R$ %.2f, %d itens faltando",
		sem.CustoLote, sem.CustoCompra, sem.ItensFaltando)
}
