// ============================================================
//  COMEÇO DE TUDO: o programa que liga o "servidor"
// ============================================================
//  Pense no sistema como um restaurante:
//    - o FRONTEND (a tela no navegador) é o salão, onde o cliente faz o pedido;
//    - o BACKEND (esta pasta, em Go) é a cozinha, que recebe o pedido,
//      faz as contas e guarda tudo;
//    - o ARQUIVO DE DADOS (data/boxcubo.json) é a despensa.
//  Este arquivo é o "abrir a cozinha": ele lê as configurações, abre o
//  arquivo de dados, atualiza dados antigos (se precisar) e fica esperando
//  pedidos da tela no endereço http://localhost:8080.

// Servidor da API de SKU da BoxCubo.
//
//	go run ./cmd/server
//
// Variáveis de ambiente:
//
//	ADDR       endereço do servidor (padrão :8080)
//	DATA_FILE  arquivo de dados local (padrão data/boxcubo.json)
//	IMG_DIR    pasta das fotos das peças (padrão data/imagens)
package main

import (
	"log"
	"net/http"
	"os"

	"boxcubo-sku/internal/api"
	"boxcubo-sku/internal/store"
)

// main é o que roda quando você digita "go run ./cmd/server".
// Passo a passo: 1) lê as configurações; 2) abre a "despensa" (arquivo de
// dados); 3) roda as atualizações de dados pendentes; 4) liga o servidor.
func main() {
	addr := env("ADDR", ":8080")
	dataFile := env("DATA_FILE", "data/boxcubo.json")
	imgDir := env("IMG_DIR", "data/imagens")

	// Troque aqui pela implementação do banco escolhido (Postgres, MySQL...).
	// Abre o arquivo de dados. Se ele ainda não existe (primeira vez),
	// é criado já com as 123 peças da V2 que vieram da planilha.
	s, err := store.NewFileStore(dataFile)
	if err != nil {
		log.Fatalf("abrindo dados: %v", err)
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

	log.Printf("BoxCubo SKU API em http://localhost%s (dados em %s)", addr, dataFile)
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
