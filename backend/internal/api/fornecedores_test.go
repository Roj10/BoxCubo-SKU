// TESTES DA PÁGINA DE FORNECEDORES: resumo, peças, código e renomear/juntar.
package api

import (
	"net/url"
	"testing"

	"boxcubo-sku/internal/model"
)

func TestFornecedores(t *testing.T) {
	h := setup(t)
	var res []resumoFornecedor
	do(t, h, "GET", "/api/fornecedores/resumo", nil, &res)
	if len(res) < 10 || res[0].Pecas < res[len(res)-1].Pecas {
		t.Fatalf("resumo: %+v", res)
	}
	var fornell []pecaFornecedor
	if c := do(t, h, "GET", "/api/fornecedores/pecas?nome=Fornell", nil, &fornell); c != 200 || len(fornell) == 0 {
		t.Fatalf("peças da Fornell: %d %d", c, len(fornell))
	}
	// "onde é usado" vem junto (para a pesquisa)
	comOnde := 0
	for _, l := range fornell {
		if len(l.OndeUsado) > 0 {
			comOnde++
		}
	}
	if comOnde == 0 {
		t.Fatal("peças da Fornell deveriam trazer onde_usado")
	}
	// código do item editado direto
	var p model.Part
	if c := do(t, h, "PUT", "/api/fornecedores/codigo", map[string]any{
		"sku": "ELE-RES-10", "fornecedor": "Fornell", "codigo": " RES10R ",
	}, &p); c != 200 || p.Fornecedores[0].Codigo != "RES10R" {
		t.Fatalf("código: %d %+v", c, p.Fornecedores)
	}
	// renomear: junta "R&A Parafusos" dentro de "1001 Parafusos" (as duas em várias peças)
	var r map[string]any
	if c := do(t, h, "POST", "/api/fornecedores/renomear", map[string]any{"de": "R&A Parafusos", "para": "1001 Parafusos"}, &r); c != 200 {
		t.Fatalf("renomear: %d", c)
	}
	par := getPart(t, h, "EST-PAR-FR38X4")
	if len(par.Fornecedores) != 1 || par.Fornecedores[0] != (model.Fornecimento{Nome: "1001 Parafusos", Codigo: "2241"}) {
		t.Fatalf("juntar duplicado: %+v", par.Fornecedores)
	}
	if c := do(t, h, "GET", "/api/fornecedores/pecas?nome="+url.QueryEscape("R&A Parafusos"), nil, nil); c != 404 {
		t.Fatalf("R&A deveria ter sumido: %d", c)
	}
	// pedido: com mínimo de 30 V2, a balança (Trentim, principal) precisa de 6
	do(t, h, "PUT", "/api/machines/v2/minimo", map[string]any{"maquinas": 30}, nil)
	var trentim []pecaFornecedor
	do(t, h, "GET", "/api/fornecedores/pecas?nome=Trentim", nil, &trentim)
	for _, l := range trentim {
		if l.SKU == "ELE-PES-BALANCA" && (l.Comprar != 6 || !l.Principal) {
			t.Fatalf("balança: %+v", l)
		}
	}
}

func TestSubstituirFornecedor(t *testing.T) {
	h := setup(t)
	// "Trentim" sai: as peças vão para "Balanças SP", sem o código antigo
	var r struct {
		Pecas         int      `json:"pecas"`
		SemFornecedor []string `json:"sem_fornecedor"`
	}
	if c := do(t, h, "POST", "/api/fornecedores/substituir", map[string]any{"de": "Trentim", "para": "Balanças SP"}, &r); c != 200 || r.Pecas == 0 {
		t.Fatalf("substituir: %d %+v", c, r)
	}
	b := getPart(t, h, "ELE-PES-BALANCA")
	if b.Fornecedores[0] != (model.Fornecimento{Nome: "Balanças SP"}) {
		t.Fatalf("balança: %+v", b.Fornecedores)
	}
	// só tirar: a balança fica sem fornecedor e aparece na lista "sem fornecedor"
	do(t, h, "POST", "/api/fornecedores/substituir", map[string]any{"de": "Balanças SP", "para": ""}, &r)
	if !contem(r.SemFornecedor, "ELE-PES-BALANCA") {
		t.Fatalf("sem fornecedor: %+v", r)
	}
	var sem []pecaFornecedor
	do(t, h, "GET", "/api/fornecedores/sem-fornecedor", nil, &sem)
	achou := false
	for _, p := range sem {
		achou = achou || p.SKU == "ELE-PES-BALANCA"
		if p.SKU == "ELE-SUB-CPUV2" {
			t.Fatal("subconjunto fabricado não deveria aparecer como sem fornecedor")
		}
	}
	if !achou {
		t.Fatal("balança deveria estar sem fornecedor")
	}
	// mesmo nome dos dois lados = erro
	if c := do(t, h, "POST", "/api/fornecedores/substituir", map[string]any{"de": "Fornell", "para": "Fornell"}, nil); c != 400 {
		t.Fatalf("mesmo nome: %d", c)
	}
}
