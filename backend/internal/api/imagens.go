// ============================================================
//  FOTOS DAS PEÇAS
// ============================================================
//  As fotos ficam na pasta backend/data/imagens, uma por peça, com o nome
//  do SKU (ex.: ELE-RES-10.png). A tela mostra a foto quando o mouse passa
//  por cima do SKU.

package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxImagem = 5 << 20 // 5 MB

var extPorTipo = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// routesImagens: fotos das peças, guardadas em disco (pasta imgDir).
// Endereços: enviar foto, remover foto e "servir" (mostrar) as fotos.
func (h *handler) routesImagens(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/parts/{sku}/imagem", h.uploadImagem)
	mux.HandleFunc("DELETE /api/parts/{sku}/imagem", h.removerImagem)
	files := http.StripPrefix("/api/imagens/", http.FileServer(http.Dir(h.imgDir)))
	mux.HandleFunc("GET /api/imagens/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		files.ServeHTTP(w, r)
	})
}

// uploadImagem recebe multipart/form-data com o campo "arquivo".
// Recebe a foto enviada pela janela da peça. Confere:
//   - tamanho (até 5 MB);
//   - se é imagem de verdade (JPG, PNG, WEBP ou GIF), olhando o conteúdo
//     do arquivo e não só o nome.
//
// Grava na pasta e guarda o endereço na peça. O "?v=número" no endereço
// obriga o navegador a mostrar a foto nova, e não uma cópia antiga.
func (h *handler) uploadImagem(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if _, err := h.s.GetPart(sku); err != nil {
		writeErr(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImagem+1024)
	f, _, err := r.FormFile("arquivo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "envie a imagem no campo 'arquivo' (até 5 MB)"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "imagem grande demais (máx. 5 MB)"})
		return
	}
	ext, ok := extPorTipo[http.DetectContentType(data)]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"erro": "formato não suportado: use JPG, PNG, WEBP ou GIF"})
		return
	}
	if err := os.MkdirAll(h.imgDir, 0o755); err != nil {
		writeErr(w, err)
		return
	}
	h.apagarArquivos(sku) // troca de formato não deixa arquivo velho
	nome := nomeArquivo(sku) + ext
	if err := os.WriteFile(filepath.Join(h.imgDir, nome), data, 0o644); err != nil {
		writeErr(w, err)
		return
	}
	// ?v= força o navegador a buscar a foto nova
	p, err := h.s.SetPartImage(sku, fmt.Sprintf("/api/imagens/%s?v=%d", nome, time.Now().Unix()))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Botão "Remover" da foto.
func (h *handler) removerImagem(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	p, err := h.s.SetPartImage(sku, "")
	if err != nil {
		writeErr(w, err)
		return
	}
	h.apagarArquivos(sku)
	writeJSON(w, http.StatusOK, p)
}

func (h *handler) apagarArquivos(sku string) {
	for _, ext := range extPorTipo {
		_ = os.Remove(filepath.Join(h.imgDir, nomeArquivo(sku)+ext))
	}
}

// nomeArquivo deixa só caracteres seguros (o SKU já é BC-XXX-YYY-NNN).
// Garante que o nome do arquivo só tenha caracteres seguros.
func nomeArquivo(sku string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, sku)
}
