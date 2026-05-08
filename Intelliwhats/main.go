package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()

	// Criar diretório temporário
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório temporário: %v", err)
	}

	// Configurar banco de dados
	dbPath := getenv("DB_PATH", DefaultDBPath)
	container, database, err := setupDatabase(ctx, dbPath)
	if err != nil {
		log.Fatalf("Erro ao configurar banco de dados: %v", err)
	}
	dbContainer = container // Definir container global
	db = database           // Definir banco global

	// Inicializar cliente WhatsApp
	client, err = initializeClient(ctx, container)
	if err != nil {
		log.Fatalf("Erro ao inicializar cliente: %v", err)
	}

	// Autenticar
	isFirstLogin, err := authenticateClient(ctx, client)
	if err != nil {
		log.Fatalf("Erro na autenticação: %v", err)
	}

	if !isFirstLogin {
		log.Println("Reconectado com sucesso!")
	}

	log.Println("Bot está pronto!")

	// Aguardar sinal de encerramento
	waitForShutdownSignal()

	log.Println("Encerrando...")
	client.Disconnect()
	db.Close()
}

func waitForShutdownSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
