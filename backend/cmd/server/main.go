// ============================================================
//  COMEÇO DE TUDO: o programa que liga o "servidor"
// ============================================================
//  Pense no sistema como um restaurante:
//    - o FRONTEND (a tela no navegador) é o salão, onde o cliente faz o pedido;
//    - o BACKEND (esta pasta, em Go) é a cozinha, que recebe o pedido,
//      faz as contas e guarda tudo;
//    - o BANCO DE DADOS (Supabase, na nuvem) é a despensa. Sem o Supabase
//      configurado, a despensa é o arquivo data/boxcubo.json.
//  Este arquivo é o "abrir a cozinha": ele lê as configurações, abre o
//  banco (ou o arquivo), atualiza dados antigos (se precisar) e fica
//  esperando pedidos da tela no endereço http://localhost:8080.

// Servidor da API de SKU da BoxCubo.
//
//	go run ./cmd/server
//
// Variáveis de ambiente:
//
//	ADDR                 endereço do servidor (padrão :8080)
//	SUPABASE_URL         endereço do projeto (https://xxxx.supabase.co)
//	SUPABASE_SECRET_KEY  chave secreta do projeto (sb_secret_...)
//	DATA_FILE            arquivo de dados local (padrão data/boxcubo.json);
//	                     com o Supabase, só é usado na 1ª vez, para copiar
//	                     os dados para o banco
//	IMG_DIR              pasta das fotos das peças (padrão data/imagens)
//
// As configurações podem ficar no arquivo backend/.env (uma por linha,
// NOME=valor). Esse arquivo não vai para o GitHub.
package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"boxcubo-sku/internal/api"
	"boxcubo-sku/internal/store"
)

// main é o que roda quando você digita "go run ./cmd/server".
// Passo a passo: 1) lê as configurações; 2) abre a "despensa" (arquivo de
// dados); 3) roda as atualizações de dados pendentes; 4) liga o servidor.
func main() {
	carregarEnv(".env")
	addr := env("ADDR", ":8080")
	dataFile := env("DATA_FILE", "data/boxcubo.json")
	imgDir := env("IMG_DIR", "data/imagens")

	// Onde ficam os dados: no Supabase (se configurado) ou no arquivo local.
	sbURL := env("SUPABASE_URL", os.Getenv("NEXT_PUBLIC_SUPABASE_URL"))
	sbChave := env("SUPABASE_SECRET_KEY", os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	var (
		s         *store.FileStore
		err       error
		ondeDados string
	)
	switch {
	case sbURL != "" && sbChave != "":
		var origem string
		s, origem, err = store.NewSupabaseStore(sbURL, sbChave, dataFile)
		if err != nil {
			log.Fatalf("abrindo o Supabase: %v", err)
		}
		log.Printf("Supabase: %s", origem)
		ondeDados = "Supabase " + sbURL
	default:
		if sbURL != "" {
			log.Printf("AVISO: SUPABASE_URL está configurado, mas falta SUPABASE_SECRET_KEY no arquivo .env — usando o arquivo local %s", dataFile)
		}
		// Abre o arquivo de dados. Se ele ainda não existe (primeira vez),
		// é criado já com as 123 peças da V2 que vieram da planilha.
		s, err = store.NewFileStore(dataFile)
		if err != nil {
			log.Fatalf("abrindo dados: %v", err)
		}
		ondeDados = dataFile
	}
	// "Migrações" são atualizações automáticas dos dados antigos. Ex.: quando os
	// SKUs mudaram de BC-ELE-RES-001 para ELE-RES-10, foi aqui que os dados de
	// vocês foram convertidos. Cada migração roda uma única vez.
	msgs, err := s.Migrar(imgDir)
	if err != nil {
		log.Fatalf("migrando dados: %v", err)
	}
	for _, m := range msgs {
		log.Printf("Dados atualizados — %s", m)
	}

	log.Printf("BoxCubo SKU API em http://localhost%s (dados em %s)", addr, ondeDados)
	// Liga o servidor e fica esperando pedidos da tela para sempre
	// (até você apertar Ctrl + C no terminal).
	log.Fatal(http.ListenAndServe(addr, api.New(s, imgDir)))
}

// env lê uma configuração do computador; se ela não existir, usa o valor
// padrão. Ex.: env("ADDR", ":8080") -> usa a porta 8080 se ninguém mudar.
func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// carregarEnv lê o arquivo .env (NOME=valor, uma por linha; linhas com #
// são comentários). O que já estiver configurado no computador vale mais.
func carregarEnv(caminho string) {
	f, err := os.Open(caminho)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		linha := strings.TrimSpace(sc.Text())
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		k, v, ok := strings.Cut(linha, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(strings.TrimPrefix(k, "export "))
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}
