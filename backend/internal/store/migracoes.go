// ============================================================
//  MIGRAÇÕES: atualizações automáticas de dados antigos
// ============================================================
//  Quando o sistema muda o jeito de guardar alguma coisa (ex.: os SKUs
//  perderam o "BC-"), os dados que já existem no computador de vocês
//  precisam ser convertidos. Cada conversão aqui é uma "migração".
//
//  Funciona como um checklist: cada migração tem um nome; depois de rodar,
//  o nome fica anotado no arquivo de dados ("migracoes") e ela nunca mais
//  roda de novo. Elas rodam sozinhas toda vez que o backend liga.

package store

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"boxcubo-sku/internal/model"
)

//go:embed sku_descritivo.json
var skuDescritivoJSON []byte

// migracao altera o arquivo de dados uma única vez (o nome fica gravado em
// "migracoes"). Rodam em ordem, com o lock já pego.
type migracao struct {
	nome   string
	aplica func(s *FileStore, imgDir string) (string, error)
}

// A lista, em ordem, de todas as conversões que o sistema já teve.
var migracoes = []migracao{
	{"sku-descritivo-v1", migrarSKUDescritivo},
	{"sem-bc-v1", migrarSemBC},
	{"fornecedores-v1", migrarFornecedores},
}

// Migrar aplica as migrações que ainda não rodaram neste arquivo e devolve
// uma frase para cada uma que mudou alguma coisa.
// Roda as migrações que ainda não rodaram e devolve uma frase para cada
// uma que mudou algo (é o que aparece no terminal: "Dados atualizados — ...").
func (s *FileStore) Migrar(imgDir string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	feitas := map[string]bool{}
	for _, m := range s.d.Migracoes {
		feitas[m] = true
	}
	msgs := []string{}
	mudou := false
	for _, m := range migracoes {
		if feitas[m.nome] {
			continue
		}
		msg, err := m.aplica(s, imgDir)
		if err != nil {
			return msgs, fmt.Errorf("migração %s: %w", m.nome, err)
		}
		if msg != "" {
			msgs = append(msgs, msg)
		}
		s.d.Migracoes = append(s.d.Migracoes, m.nome)
		mudou = true
	}
	if mudou {
		return msgs, s.save()
	}
	return msgs, nil
}

// SKUs numerados da V2 -> finais descritivos (BC-ELE-RES-001 -> BC-ELE-RES-10).
// Migração 1: troca os SKUs numerados pelos descritivos usando a tabela
// sku_descritivo.json (ex.: BC-ELE-RES-001 -> BC-ELE-RES-10).
func migrarSKUDescritivo(s *FileStore, imgDir string) (string, error) {
	var pares [][2]string
	if err := json.Unmarshal(skuDescritivoJSON, &pares); err != nil {
		return "", err
	}
	n := 0
	for _, par := range pares {
		antigo, novo := par[0], par[1]
		if s.partIndex(antigo) < 0 || s.partIndex(novo) >= 0 {
			continue // já migrado, apagado ou com conflito: não mexe
		}
		if _, err := s.renameSKU(antigo, novo); err != nil {
			return "", err
		}
		RenomearArquivosImagem(imgDir, antigo, novo)
		n++
	}
	if n == 0 {
		return "", nil
	}
	return fmt.Sprintf("SKUs com final descritivo: %d peças", n), nil
}

// Tira o "BC-" do começo dos SKUs (BC-ELE-RES-10 -> ELE-RES-10, BC-MAQ-V2 -> MAQ-V2).
// Migração 2: tira o "BC-" do começo dos códigos
// (BC-ELE-RES-10 -> ELE-RES-10 e BC-MAQ-V2 -> MAQ-V2).
func migrarSemBC(s *FileStore, imgDir string) (string, error) {
	n := 0
	for _, p := range append([]model.Part(nil), s.d.Parts...) {
		if !strings.HasPrefix(p.SKU, "BC-") {
			continue
		}
		novo := strings.TrimPrefix(p.SKU, "BC-")
		if s.partIndex(novo) >= 0 {
			continue
		}
		if _, err := s.renameSKU(p.SKU, novo); err != nil {
			return "", err
		}
		RenomearArquivosImagem(imgDir, p.SKU, novo)
		n++
	}
	for i := range s.d.Machines {
		s.d.Machines[i].SKU = strings.TrimPrefix(s.d.Machines[i].SKU, "BC-")
	}
	if n == 0 {
		return "", nil
	}
	return fmt.Sprintf("SKUs sem o \"BC-\": %d peças", n), nil
}

// Converte o texto livre "FORNELL / ML" numa lista de fornecedores com código.
// Migração 3: o fornecedor era um texto solto ("FORNELL / ML"); agora vira
// uma lista organizada, com espaço para o código do item em cada um.
func migrarFornecedores(s *FileStore, _ string) (string, error) {
	n := 0
	for i := range s.d.Parts {
		p := &s.d.Parts[i]
		if len(p.Fornecedores) == 0 && strings.TrimSpace(p.FornecedorLegado) != "" {
			p.Fornecedores = FornecedoresDeTexto(p.FornecedorLegado, p.Nome)
			n++
		}
		p.FornecedorLegado = ""
		if p.Fornecedores == nil {
			p.Fornecedores = []model.Fornecimento{}
		}
	}
	if n == 0 {
		return "", nil
	}
	return fmt.Sprintf("Fornecedores separados com código: %d peças", n), nil
}

// nomes padronizados dos fornecedores que aparecem na planilha
// Tabela para padronizar os nomes que vinham escritos de jeitos diferentes
// na planilha (ex.: "ML" -> "Mercado Livre", "FORNELL" -> "Fornell").
var nomesFornecedor = map[string]string{
	"fornell":                 "Fornell",
	"ml":                      "Mercado Livre",
	"mercado livre":           "Mercado Livre",
	"1001 parafusos":          "1001 Parafusos",
	"r&a parafusos":           "R&A Parafusos",
	"abu mazin":               "Abu Mazin",
	"wincabos":                "Wincabos",
	"griffus":                 "Griffus",
	"trentim":                 "Trentim",
	"dominik":                 "Dominik",
	"gr":                      "GR",
	"metalporto":              "MetalPorto",
	"zinca rapido":            "Zinca Rápido",
	"tecnograph":              "Tecnograph",
	"sombra print":            "Sombra Print",
	"patola":                  "Patola",
	"trombini":                "Trombini",
	"lri mw":                  "LRI MW",
	"kabum":                   "Kabum",
	"bite":                    "Bite",
	"loja completa":           "Loja Completa",
	"casa das borrachas":      "Casa das Borrachas",
	"mercadão das embalagens": "Mercadão das Embalagens",
	"cia embalagem":           "Cia Embalagem",
}

var (
	reURL          = regexp.MustCompile(`:?\s*https?://\S+`)
	reCodigoInicio = regexp.MustCompile(`^(\d{3,6})\s`) // "2241 Parafuso Frances..." (1001 Parafusos)
	reCodigoParen  = regexp.MustCompile(`\((\d{5,})\)`) // "Caixa de papelão (398737)" (Trombini)
)

// FornecedoresDeTexto separa "Dominik / MetalPorto" em itens e tenta achar o
// código do item no fornecedor dentro do nome da peça.
// Quebra o texto antigo em fornecedores e tenta achar o código do item
// escondido no nome da peça. Ex.: na peça "2241 Parafuso Frances...", o
// "2241" é o código da 1001 Parafusos; na "Caixa de papelão (398737)",
// o "398737" é o código da Trombini.
func FornecedoresDeTexto(txt, nomePeca string) []model.Fornecimento {
	txt = reURL.ReplaceAllString(txt, "")
	out := []model.Fornecimento{}
	vistos := map[string]bool{}
	for _, parte := range strings.Split(txt, "/") {
		nome := strings.TrimSpace(parte)
		if nome == "" {
			continue
		}
		if padrao, ok := nomesFornecedor[strings.ToLower(nome)]; ok {
			nome = padrao
		}
		if vistos[nome] {
			continue
		}
		vistos[nome] = true
		f := model.Fornecimento{Nome: nome}
		switch nome {
		case "1001 Parafusos":
			if m := reCodigoInicio.FindStringSubmatch(nomePeca); m != nil {
				f.Codigo = m[1]
			}
		case "Trombini":
			if m := reCodigoParen.FindStringSubmatch(nomePeca); m != nil {
				f.Codigo = m[1]
			}
		}
		out = append(out, f)
	}
	return out
}
