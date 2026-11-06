// TESTES DO SKU E FORNECEDORES: final escolhido, troca de código,
// conversão de dados antigos (BC-, numerados, fornecedor em texto) e
// fornecedores com código.

package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boxcubo-sku/internal/model"
	"boxcubo-sku/internal/store"
)

func TestSufixoEscolhidoERenomear(t *testing.T) {
	h := setup(t)
	var p model.Part
	if c := do(t, h, "POST", "/api/parts", map[string]any{
		"prefixo": "ELE-RES", "sufixo": "47", "nome": "47 Ω",
	}, &p); c != 201 || p.SKU != "ELE-RES-47" {
		t.Fatalf("criar com final: %d %+v", c, p)
	}
	// final repetido
	if c := do(t, h, "POST", "/api/parts", map[string]any{"prefixo": "ELE-RES", "sufixo": "47", "nome": "x"}, nil); c != 409 {
		t.Fatalf("final repetido deveria dar 409, veio %d", c)
	}
	// final inválido
	if c := do(t, h, "POST", "/api/parts", map[string]any{"prefixo": "ELE-RES", "sufixo": "4-7", "nome": "x"}, nil); c != 400 {
		t.Fatalf("final com hífen deveria dar 400, veio %d", c)
	}
	// renomear peça que está na BOM e em movimentações
	do(t, h, "POST", "/api/saidas", map[string]any{
		"itens": []map[string]any{{"sku": "ELE-RES-10K", "quantidade": 3}},
	}, nil)
	var r model.Part
	if c := do(t, h, "PUT", "/api/parts/ELE-RES-10K/sku", map[string]any{"sufixo": "10k0"}, &r); c != 200 || r.SKU != "ELE-RES-10K0" {
		t.Fatalf("renomear: %d %+v", c, r)
	}
	var bom bomResponse
	do(t, h, "GET", "/api/machines/v2/bom", nil, &bom)
	achou := false
	for _, l := range bom.Linhas {
		if l.SKU == "ELE-RES-10K" {
			t.Fatal("SKU antigo ficou na BOM")
		}
		achou = achou || l.SKU == "ELE-RES-10K0"
	}
	if !achou {
		t.Fatal("SKU novo não está na BOM")
	}
	var movs []model.Movement
	do(t, h, "GET", "/api/movements", nil, &movs)
	if movs[0].Itens[0].SKU != "ELE-RES-10K0" {
		t.Fatalf("movimentação não acompanhou: %+v", movs[0].Itens)
	}
	// renomear para um que já existe
	if c := do(t, h, "PUT", "/api/parts/ELE-RES-10K0/sku", map[string]any{"sufixo": "47"}, nil); c != 409 {
		t.Fatalf("renomear para existente deveria dar 409, veio %d", c)
	}
}

// Dados antigos (SKUs numerados com "BC-", fornecedor em texto livre) são
// convertidos: final descritivo, sem "BC-" e fornecedores com código.
func TestMigracaoDadosAntigos(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "db.json")
	imgDir := filepath.Join(dir, "imagens")

	// monta um arquivo "antigo" a partir do seed, voltando os SKUs para BC-...-001
	var pares [][2]string
	tabela, _ := os.ReadFile("../store/sku_descritivo.json")
	json.Unmarshal(tabela, &pares)
	raw, _ := os.ReadFile("../store/seed.json")
	txt := string(raw)
	for _, p := range pares {
		txt = strings.ReplaceAll(txt, `"`+strings.TrimPrefix(p[1], "BC-")+`"`, `"`+p[0]+`"`)
	}
	txt = strings.ReplaceAll(txt, `"MAQ-`, `"BC-MAQ-`)
	os.WriteFile(db, []byte(txt), 0o644)
	os.MkdirAll(imgDir, 0o755)
	os.WriteFile(filepath.Join(imgDir, "BC-ELE-PES-001.png"), pngMinimo, 0o644)

	s, err := store.NewFileStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetPart("BC-ELE-RES-001"); err != nil {
		t.Fatal("o arquivo de teste deveria ter o SKU antigo")
	}
	s.SetPartImage("BC-ELE-PES-001", "/api/imagens/BC-ELE-PES-001.png?v=1")

	msgs, err := s.Migrar(imgDir)
	if err != nil || len(msgs) != 3 {
		t.Fatalf("migrar: %v %v", msgs, err)
	}
	t.Log(msgs)
	p, err := s.GetPart("ELE-PES-BALANCA")
	if err != nil || p.Imagem != "/api/imagens/ELE-PES-BALANCA.png?v=1" {
		t.Fatalf("balança: %+v %v", p, err)
	}
	if _, err := os.Stat(filepath.Join(imgDir, "ELE-PES-BALANCA.png")); err != nil {
		t.Fatal("arquivo da foto não foi renomeado")
	}
	if len(p.Fornecedores) != 1 || p.Fornecedores[0].Nome != "Trentim" {
		t.Fatalf("fornecedor da balança: %+v", p.Fornecedores)
	}
	bom, _ := s.GetBOM("v2")
	for _, it := range bom {
		if strings.HasPrefix(it.SKU, "BC-") || strings.HasSuffix(it.SKU, "-001") {
			t.Fatalf("BOM ainda tem SKU antigo: %s", it.SKU)
		}
	}
	if m, _ := s.GetMachine("v2"); m.SKU != "MAQ-V2" {
		t.Fatalf("máquina: %s", m.SKU)
	}
	// parafuso da 1001 Parafusos: código separado do nome
	par, _ := s.GetPart("EST-PAR-FR38X4")
	if len(par.Fornecedores) != 2 || par.Fornecedores[0] != (model.Fornecimento{Nome: "1001 Parafusos", Codigo: "2241"}) ||
		par.Fornecedores[1].Nome != "R&A Parafusos" {
		t.Fatalf("parafuso: %+v", par.Fornecedores)
	}
	cx, _ := s.GetPart("EMB-EMB-CAIXA")
	if cx.Fornecedores[0] != (model.Fornecimento{Nome: "Trombini", Codigo: "398737"}) {
		t.Fatalf("caixa: %+v", cx.Fornecedores)
	}
	// roda uma vez só
	if msgs, _ := s.Migrar(imgDir); len(msgs) != 0 {
		t.Fatalf("migração rodou de novo: %v", msgs)
	}
}

func TestFornecedoresComCodigo(t *testing.T) {
	h := setup(t)
	var p model.Part
	do(t, h, "GET", "/api/parts/ELE-CTR-RPI3B", nil, &p)
	if len(p.Fornecedores) != 1 || p.Fornecedores[0].Nome != "Mercado Livre" {
		t.Fatalf("seed: %+v", p.Fornecedores)
	}
	p.Fornecedores = []model.Fornecimento{{Nome: "Mercado Livre", Codigo: "MLB-123"}, {Nome: "Filipeflop", Codigo: "RPI-3B"}, {Nome: " ", Codigo: ""}}
	var up model.Part
	do(t, h, "PUT", "/api/parts/ELE-CTR-RPI3B", p, &up)
	if len(up.Fornecedores) != 2 || up.Fornecedores[1].Codigo != "RPI-3B" {
		t.Fatalf("salvar: %+v", up.Fornecedores)
	}
	// busca pelo código do fornecedor
	var achou []model.Part
	do(t, h, "GET", "/api/parts?q=mlb-123", nil, &achou)
	if len(achou) != 1 || achou[0].SKU != "ELE-CTR-RPI3B" {
		t.Fatalf("busca: %+v", achou)
	}
	// aparece no planejamento
	var plan model.Plan
	do(t, h, "GET", "/api/machines/v2/plan?quantidade=1", nil, &plan)
	for _, l := range plan.Linhas {
		if l.SKU == "ELE-CTR-RPI3B" && (len(l.Fornecedores) != 2 || l.Fornecedores[0].Codigo != "MLB-123") {
			t.Fatalf("plano: %+v", l.Fornecedores)
		}
	}
	var nomes []string
	do(t, h, "GET", "/api/fornecedores", nil, &nomes)
	if len(nomes) < 10 {
		t.Fatalf("fornecedores: %v", nomes)
	}
}
