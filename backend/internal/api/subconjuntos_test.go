// TESTES DOS SUBCONJUNTOS E DAS RECEITAS DO CONTROLE: CPU e Shield com
// receita própria, custo somado, "usada em", fabricação, planejamento que
// desdobra o subconjunto e estoque mínimo das peças de dentro dele.
package api

import (
	"strings"
	"testing"

	"boxcubo-sku/internal/model"
)

func TestReceitasDoControle(t *testing.T) {
	h := setup(t)
	var subs []model.Machine
	do(t, h, "GET", "/api/subconjuntos", nil, &subs)
	if len(subs) != 2 {
		t.Fatalf("subconjuntos: %+v", subs)
	}
	for _, id := range []string{"vision", "eye"} {
		var bom bomResponse
		do(t, h, "GET", "/api/machines/"+id+"/bom", nil, &bom)
		if !bom.Machine.Ativa || len(bom.Linhas) < 20 {
			t.Fatalf("%s: ativa=%v linhas=%d", id, bom.Machine.Ativa, len(bom.Linhas))
		}
	}
	// V2.5, Dynamic e Slim foram retiradas: só ficam as três máquinas
	var maquinas []model.Machine
	do(t, h, "GET", "/api/machines", nil, &maquinas)
	ids := []string{}
	for _, m := range maquinas {
		ids = append(ids, m.ID)
	}
	if strings.Join(ids, ",") != "v2,vision,eye" {
		t.Fatalf("máquinas: %v", ids)
	}
	if c := do(t, h, "GET", "/api/machines/slim", nil, nil); c != 404 {
		t.Fatalf("slim deveria ter saído: %d", c)
	}
	// receita da CPU e custo somado
	var cpu bomResponse
	do(t, h, "GET", "/api/machines/ELE-SUB-CPUV2/bom", nil, &cpu)
	if !cpu.Machine.Subconjunto || len(cpu.Linhas) != 10 || cpu.Peca == nil {
		t.Fatalf("cpu: %+v", cpu.Machine)
	}
	if cpu.CustoPadrao < 526 { // tem pelo menos o Raspberry (R$ 526,20)
		t.Fatalf("custo da CPU: %v", cpu.CustoPadrao)
	}
	// na V2 a CPU aparece com custo calculado e "usada em"
	var v2 bomResponse
	do(t, h, "GET", "/api/machines/v2/bom", nil, &v2)
	var linhaCPU, linhaMonitor *model.BOMLine
	for i, l := range v2.Linhas {
		switch l.SKU {
		case "ELE-SUB-CPUV2":
			linhaCPU = &v2.Linhas[i]
		case "ELE-PER-MONITOR15":
			linhaMonitor = &v2.Linhas[i]
		}
	}
	if linhaCPU == nil || linhaCPU.CustoUnitario == nil || *linhaCPU.CustoUnitario != cpu.CustoPadrao {
		t.Fatalf("CPU na V2: %+v", linhaCPU)
	}
	if linhaMonitor == nil || len(linhaMonitor.UsadaEm) != 3 {
		t.Fatalf("monitor deveria ser usado em 3 máquinas: %+v", linhaMonitor)
	}
	var shield bomResponse
	do(t, h, "GET", "/api/machines/ELE-PCB-SHIELDRPI/bom", nil, &shield)
	for _, l := range shield.Linhas {
		if l.SKU == "ELE-RES-10" && (len(l.UsadaEm) != 1 || l.UsadaEm[0] != "Estática V2") {
			t.Fatalf("resistor 10 Ω (dentro do Shield) usado em: %v", l.UsadaEm)
		}
	}
	var cpuNaEye bomResponse
	do(t, h, "GET", "/api/machines/eye/bom", nil, &cpuNaEye)
	for _, l := range cpuNaEye.Linhas {
		if l.SKU == "ELE-CTR-RPI3B" && len(l.UsadaEm) != 2 {
			t.Fatalf("Raspberry: EYE direto + V2 dentro da CPU: %v", l.UsadaEm)
		}
	}
}

func TestFabricarSubconjunto(t *testing.T) {
	h := setup(t)
	// Shield Raspberry está com estoque 0 -> fabricar CPU avisa falta
	var aviso struct {
		Faltas []model.Shortage `json:"faltas"`
	}
	if c := do(t, h, "POST", "/api/parts/ELE-SUB-CPUV2/fabricar", map[string]any{"quantidade": 2}, &aviso); c != 409 {
		t.Fatalf("deveria avisar falta, veio %d", c)
	}
	// fabrica 2 Shields primeiro (Trimpot está em 0 -> continuar mesmo assim)
	var m1 movResp
	if c := do(t, h, "POST", "/api/parts/ELE-PCB-SHIELDRPI/fabricar", map[string]any{"quantidade": 2, "permitir_falta": true}, &m1); c != 201 || m1.Tipo != model.MovFabricacao {
		t.Fatalf("fabricar shield: %d %+v", c, m1.Movement)
	}
	if p := getPart(t, h, "ELE-PCB-SHIELDRPI"); p.Estoque != 2 {
		t.Fatalf("shield = %v", p.Estoque)
	}
	if p := getPart(t, h, "ELE-RES-100"); p.Estoque != 1342-6 { // 3 por shield
		t.Fatalf("resistor 100 = %v", p.Estoque)
	}
	// agora 2 CPUs (barra de pino 180° está em 0 -> continuar)
	do(t, h, "POST", "/api/parts/ELE-SUB-CPUV2/fabricar", map[string]any{"quantidade": 2, "permitir_falta": true}, nil)
	if p := getPart(t, h, "ELE-SUB-CPUV2"); p.Estoque != 2 {
		t.Fatalf("cpu = %v", p.Estoque)
	}
	if p := getPart(t, h, "ELE-CTR-RPI3B"); p.Estoque != 23 {
		t.Fatalf("raspberry = %v", p.Estoque)
	}
	if p := getPart(t, h, "ELE-PCB-SHIELDRPI"); p.Estoque != 0 {
		t.Fatalf("shields usados = %v", p.Estoque)
	}
	// peça comum não pode ser "fabricada"
	if c := do(t, h, "POST", "/api/parts/ELE-RES-10/fabricar", map[string]any{"quantidade": 1}, nil); c != 400 {
		t.Fatalf("fabricar peça comum: %d", c)
	}
	// receita circular é recusada
	if c := do(t, h, "PUT", "/api/machines/ELE-PCB-SHIELDRPI/bom/ELE-SUB-CPUV2", map[string]any{"quantidade": 1}, nil); c != 400 {
		t.Fatalf("receita circular: %d", c)
	}
}

func TestPlanoDesdobraSubconjunto(t *testing.T) {
	h := setup(t)
	var plan model.Plan
	do(t, h, "GET", "/api/machines/v2/plan?quantidade=1&margem=0", nil, &plan)
	linhas := map[string]model.PlanLine{}
	for _, l := range plan.Linhas {
		linhas[l.SKU] = l
	}
	cpu := linhas["ELE-SUB-CPUV2"]
	if !cpu.Fabricar || cpu.Comprar != 1 || cpu.CustoCompra != nil {
		t.Fatalf("cpu: %+v", cpu)
	}
	rpi := linhas["ELE-CTR-RPI3B"]
	if rpi.Necessario != 1 || len(rpi.Via) != 1 {
		t.Fatalf("raspberry via CPU: %+v", rpi)
	}
	res := linhas["ELE-RES-100"] // dentro do Shield, dentro da CPU
	if res.Necessario != 3 || res.Comprar != 0 {
		t.Fatalf("resistor 100 via Shield: %+v", res)
	}
	// mínimo: 2 máquinas -> Raspberry mínimo 2 (via CPU); a CPU em si não entra
	do(t, h, "PUT", "/api/machines/v2/minimo", map[string]any{"maquinas": 2}, nil)
	var bom bomResponse
	do(t, h, "GET", "/api/machines/ELE-SUB-CPUV2/bom", nil, &bom)
	for _, l := range bom.Linhas {
		if l.SKU == "ELE-CTR-RPI3B" && l.Minimo != 2 {
			t.Fatalf("mínimo do raspberry: %v", l.Minimo)
		}
	}
}
