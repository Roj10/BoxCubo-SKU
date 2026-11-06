// TESTES DO ESTOQUE MÍNIMO: cálculo pelo número de máquinas, kits,
// reposição e o aviso depois de uma saída.

package api

import (
	"testing"

	"boxcubo-sku/internal/model"
)

func TestEstoqueMinimo(t *testing.T) {
	h := setup(t)
	// sem mínimo configurado: nada na reposição
	var rep reposicaoResp
	do(t, h, "GET", "/api/reposicao", nil, &rep)
	if rep.Itens != 0 {
		t.Fatalf("sem mínimo não deveria ter reposição: %d", rep.Itens)
	}

	// V2: manter peças para 30 máquinas (balança: estoque 27 < 30)
	var m model.Machine
	if c := do(t, h, "PUT", "/api/machines/v2/minimo", map[string]any{"maquinas": 30}, &m); c != 200 || m.MinimoMaquinas != 30 {
		t.Fatalf("set minimo: %d %+v", c, m)
	}
	do(t, h, "GET", "/api/reposicao", nil, &rep)
	var bal *model.ReposicaoLine
	for i, l := range rep.Linhas {
		if l.SKU == "ELE-PES-BALANCA" {
			bal = &rep.Linhas[i]
		}
		if l.SKU == "ELE-PER-MONITOR15" {
			t.Fatal("monitor (kit) não deveria entrar sem com_kits")
		}
	}
	if bal == nil || bal.Minimo != 30 || bal.Sugestao != 3 {
		t.Fatalf("balança: %+v", bal)
	}
	if len(rep.Linhas) > 0 && !rep.Linhas[0].Zerado {
		t.Fatal("zerados deveriam vir primeiro")
	}

	// com kits: monitor entra (estoque 3 < 30)
	do(t, h, "PUT", "/api/machines/v2/minimo", map[string]any{"maquinas": 30, "com_kits": true}, nil)
	var rep2 reposicaoResp
	do(t, h, "GET", "/api/reposicao", nil, &rep2)
	if rep2.Itens <= rep.Itens {
		t.Fatal("com kits deveria aumentar a reposição")
	}

	// BOM mostra o mínimo e a contagem
	var bom bomResponse
	do(t, h, "GET", "/api/machines/v2/bom", nil, &bom)
	if bom.AbaixoMinimo != rep2.Itens {
		t.Fatalf("bom.abaixo_minimo=%d, reposição=%d", bom.AbaixoMinimo, rep2.Itens)
	}

	// mínimo baixo (2 máquinas): saída de 1 máquina deixa a balança? não (27-1=26 ≥ 2)
	do(t, h, "PUT", "/api/machines/v2/minimo", map[string]any{"maquinas": 2}, nil)
	var r1 movResp
	do(t, h, "POST", "/api/machines/v2/saida", map[string]any{"numeros_serie": []string{"M1"}, "permitir_falta": true}, &r1)
	for _, l := range r1.AbaixoMinimo {
		if l.SKU == "ELE-PES-BALANCA" {
			t.Fatal("balança não deveria estar abaixo do mínimo")
		}
	}
	if r1.ID == "" {
		t.Fatal("resposta da saída deveria trazer a movimentação")
	}
	// saída de peças: 25 balanças -> sobra 1 < 2 -> alerta
	var r2 movResp
	do(t, h, "POST", "/api/saidas", map[string]any{
		"itens": []map[string]any{{"sku": "ELE-PES-BALANCA", "quantidade": 25}},
	}, &r2)
	if len(r2.AbaixoMinimo) != 1 || r2.AbaixoMinimo[0].Sugestao != 1 {
		t.Fatalf("alerta após saída: %+v", r2.AbaixoMinimo)
	}
}
