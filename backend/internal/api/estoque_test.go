// TESTES DO ESTOQUE: saída de máquina, aviso de falta, "continuar mesmo
// assim" (fica em 0), estorno, número de série repetido, entrada e kits.

package api

import (
	"net/http"
	"testing"

	"boxcubo-sku/internal/model"
)

func getPart(t *testing.T, h http.Handler, sku string) model.Part {
	t.Helper()
	var p model.Part
	do(t, h, "GET", "/api/parts/"+sku, nil, &p)
	return p
}

func TestSaidaComEstoque(t *testing.T) {
	h := setup(t)
	// Balança: 1 por máquina, estoque inicial 24 (vindo do Controle)
	var mov model.Movement
	code := do(t, h, "POST", "/api/machines/v2/saida", map[string]any{
		"numeros_serie": []string{"bc-001"}, "permitir_falta": true,
	}, &mov)
	if code != 201 || mov.Quantidade != 1 || mov.NumerosSerie[0] != "BC-001" {
		t.Fatalf("saida: %d %+v", code, mov)
	}
	if p := getPart(t, h, "ELE-PES-BALANCA"); p.Estoque != 23 {
		t.Fatalf("estoque balança = %v, esperava 23", p.Estoque)
	}
	// mesmo número de série de novo -> bloqueado
	if c := do(t, h, "POST", "/api/machines/v2/saida", map[string]any{
		"numeros_serie": []string{"BC-001"}, "permitir_falta": true,
	}, nil); c != 409 {
		t.Fatalf("serial repetido deveria dar 409, veio %d", c)
	}
}

func TestSaidaSemEstoqueAvisaEDeixaEmZero(t *testing.T) {
	h := setup(t)
	// Balança: estoque 24. Tentar 30 máquinas.
	series := []string{}
	for i := 0; i < 30; i++ {
		series = append(series, "S"+string(rune('A'+i/10))+string(rune('0'+i%10)))
	}
	var aviso struct {
		Erro   string           `json:"erro"`
		Faltas []model.Shortage `json:"faltas"`
	}
	if c := do(t, h, "POST", "/api/machines/v2/saida", map[string]any{"numeros_serie": series}, &aviso); c != 409 {
		t.Fatalf("sem permitir_falta deveria dar 409, veio %d", c)
	}
	achou := false
	for _, f := range aviso.Faltas {
		if f.Nome == "Balança" && f.Falta == 6 {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("balança deveria faltar 6: %+v", aviso.Faltas)
	}
	if p := getPart(t, h, "ELE-PES-BALANCA"); p.Estoque != 24 {
		t.Fatalf("nada deveria ter saído ainda, balança = %v", p.Estoque)
	}

	// "continuar mesmo assim"
	var mov model.Movement
	do(t, h, "POST", "/api/machines/v2/saida", map[string]any{"numeros_serie": series, "permitir_falta": true}, &mov)
	if !mov.ComFalta {
		t.Fatal("movimentação deveria estar marcada com falta")
	}
	if p := getPart(t, h, "ELE-PES-BALANCA"); p.Estoque != 0 {
		t.Fatalf("balança deveria ficar em 0, ficou %v", p.Estoque)
	}

	// estorno devolve exatamente o que saiu (24, não 30)
	var est model.Movement
	if c := do(t, h, "POST", "/api/movements/"+mov.ID+"/estorno", nil, &est); c != 201 {
		t.Fatalf("estorno: %d", c)
	}
	if p := getPart(t, h, "ELE-PES-BALANCA"); p.Estoque != 24 {
		t.Fatalf("estorno deveria voltar a 24, ficou %v", p.Estoque)
	}
	if c := do(t, h, "POST", "/api/movements/"+mov.ID+"/estorno", nil, nil); c != 409 {
		t.Fatalf("estornar duas vezes deveria dar 409, veio %d", c)
	}
	// depois do estorno, os números de série voltam a ficar livres
	if c := do(t, h, "POST", "/api/machines/v2/saida", map[string]any{
		"numeros_serie": []string{series[0]}, "permitir_falta": true,
	}, nil); c != 201 {
		t.Fatalf("serial estornado deveria poder sair de novo, veio %d", c)
	}
}

func TestEntradaEKit(t *testing.T) {
	h := setup(t)
	var mov model.Movement
	if c := do(t, h, "POST", "/api/entradas", map[string]any{
		"itens": []map[string]any{{"sku": "ELE-PES-BALANCA", "quantidade": 10}},
	}, &mov); c != 201 {
		t.Fatalf("entrada: %d", c)
	}
	if p := getPart(t, h, "ELE-PES-BALANCA"); p.Estoque != 34 {
		t.Fatalf("balança = %v, esperava 34", p.Estoque)
	}
	// leitor 2D só sai quando o kit ADDC é marcado (estoque 8)
	do(t, h, "POST", "/api/machines/v2/saida", map[string]any{"numeros_serie": []string{"X1"}, "permitir_falta": true}, nil)
	if p := getPart(t, h, "ELE-PER-LEITOR2D"); p.Estoque != 8 {
		t.Fatalf("leitor não deveria sair sem kit: %v", p.Estoque)
	}
	do(t, h, "POST", "/api/machines/v2/saida", map[string]any{
		"numeros_serie": []string{"X2"}, "kits": []string{"ADDC"}, "permitir_falta": true,
	}, nil)
	if p := getPart(t, h, "ELE-PER-LEITOR2D"); p.Estoque != 7 {
		t.Fatalf("leitor deveria sair com kit: %v", p.Estoque)
	}
	var lista []model.Movement
	do(t, h, "GET", "/api/movements?q=X2", nil, &lista)
	if len(lista) != 1 || lista[0].Kits[0] != "ADDC" {
		t.Fatalf("busca por série: %+v", lista)
	}
}

func TestSaidaPecasAvulsas(t *testing.T) {
	h := setup(t)
	// envia 2 sensores (estoque 101) e 5 shields sensor (estoque 0) para manutenção
	body := map[string]any{
		"itens": []map[string]any{
			{"sku": "ELE-CMP-HYSR05", "quantidade": 2},
			{"sku": "ELE-PCB-SHIELDSENS", "quantidade": 5},
		},
		"cliente": "Correios CDD Floripa", "numero_serie": "bc2-0100",
	}
	var aviso struct {
		Faltas []model.Shortage `json:"faltas"`
	}
	if c := do(t, h, "POST", "/api/saidas", body, &aviso); c != 409 || len(aviso.Faltas) != 1 {
		t.Fatalf("deveria avisar falta do shield: %d %+v", c, aviso)
	}
	body["permitir_falta"] = true
	var mov model.Movement
	if c := do(t, h, "POST", "/api/saidas", body, &mov); c != 201 || mov.Tipo != model.MovSaidaPecas || !mov.ComFalta {
		t.Fatalf("saida: %d %+v", c, mov)
	}
	if p := getPart(t, h, "ELE-CMP-HYSR05"); p.Estoque != 99 {
		t.Fatalf("sensor = %v", p.Estoque)
	}
	// busca pela máquina do cliente
	var lista []model.Movement
	do(t, h, "GET", "/api/movements?q=BC2-0100", nil, &lista)
	if len(lista) != 1 {
		t.Fatalf("busca por série da máquina do cliente: %d", len(lista))
	}
	// o número de série da máquina do cliente não bloqueia saída de máquina nova com outro número
	if c := do(t, h, "POST", "/api/machines/v2/saida", map[string]any{"numeros_serie": []string{"BC2-0200"}, "permitir_falta": true}, nil); c != 201 {
		t.Fatalf("saida maquina: %d", c)
	}
	// estorno da saída avulsa devolve
	do(t, h, "POST", "/api/movements/"+mov.ID+"/estorno", nil, nil)
	// 99 - 10 (máquina) + 2 (estorno) = 91
	if p := getPart(t, h, "ELE-CMP-HYSR05"); p.Estoque != 91 {
		t.Fatalf("estorno não devolveu: %v", p.Estoque)
	}
}
