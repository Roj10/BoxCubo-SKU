// TESTES DAS FOTOS: enviar, mostrar, remover e recusar arquivo que não é imagem.

package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"boxcubo-sku/internal/model"
)

// PNG 1x1 válido
var pngMinimo = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

func TestImagem(t *testing.T) {
	h := setup(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("arquivo", "foto.png")
	fw.Write(pngMinimo)
	mw.Close()
	req := httptest.NewRequest("POST", "/api/parts/ELE-PES-BALANCA/imagem", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var p model.Part
	json.Unmarshal(rec.Body.Bytes(), &p)
	if rec.Code != 200 || !strings.HasPrefix(p.Imagem, "/api/imagens/ELE-PES-BALANCA.png?v=") {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	// a foto é servida
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", p.Imagem, nil))
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), pngMinimo) {
		t.Fatalf("servir imagem: %d", rec.Code)
	}
	// editar a peça não apaga a foto
	p.Nome = "Balança 100kg"
	p.Imagem = ""
	var up model.Part
	do(t, h, "PUT", "/api/parts/ELE-PES-BALANCA", p, &up)
	if up.Imagem == "" {
		t.Fatal("editar a peça apagou a foto")
	}
	// aparece no planejamento
	var plan model.Plan
	do(t, h, "GET", "/api/machines/v2/plan?quantidade=1", nil, &plan)
	achou := false
	for _, l := range plan.Linhas {
		if l.SKU == "ELE-PES-BALANCA" && l.Imagem != "" {
			achou = true
		}
	}
	if !achou {
		t.Fatal("imagem não veio no planejamento")
	}
	// remover
	var rm model.Part
	do(t, h, "DELETE", "/api/parts/ELE-PES-BALANCA/imagem", nil, &rm)
	if rm.Imagem != "" {
		t.Fatal("não removeu")
	}
	// arquivo que não é imagem
	body.Reset()
	mw = multipart.NewWriter(&body)
	fw, _ = mw.CreateFormFile("arquivo", "x.txt")
	fw.Write([]byte("não sou imagem"))
	mw.Close()
	req = httptest.NewRequest("POST", "/api/parts/ELE-PES-BALANCA/imagem", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("texto deveria ser recusado, veio %d", rec.Code)
	}
}
