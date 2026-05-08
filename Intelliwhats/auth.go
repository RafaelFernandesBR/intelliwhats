package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// setupDatabase configura e retorna o container do banco de dados
func setupDatabase(ctx context.Context, dbPath string) (*sqlstore.Container, *sql.DB, error) {
	// Criar diretório se não existir
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, nil, fmt.Errorf("criar diretório do banco: %v", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", dbPath)
	dbLog := waLog.Stdout("DB", "INFO", true)

	container, err := sqlstore.New(ctx, "sqlite3", dsn, dbLog)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlstore.New: %v", err)
	}

	// Abrir conexão direta para uso geral
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sql.Open: %v", err)
	}

	// Inicializar tabela de preferências
	if err := initPreferencesTable(ctx, db); err != nil {
		return nil, nil, fmt.Errorf("inicializar preferências: %v", err)
	}

	return container, db, nil
}

// initializeClient cria e configura o cliente WhatsApp
func initializeClient(ctx context.Context, container *sqlstore.Container) (*whatsmeow.Client, error) {
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetFirstDevice: %v", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	return client, nil
}

// authenticateClient autentica o cliente WhatsApp
// Retorna true se é primeiro login, false se já estava autenticado
func authenticateClient(ctx context.Context, client *whatsmeow.Client) (bool, error) {
	isFirstLogin := client.Store.ID == nil

	if isFirstLogin {
		// Primeiro login - pareamento via código
		phone := os.Getenv("PHONE_NUMBER")
		if phone == "" {
			return false, fmt.Errorf("PHONE_NUMBER é obrigatório para o primeiro pareamento")
		}

		log.Printf("Dispositivo não pareado, solicitando código para %s", phone)

		if err := client.Connect(); err != nil {
			return false, fmt.Errorf("connect: %v", err)
		}

		code, err := client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		if err != nil {
			return false, fmt.Errorf("PairPhone: %v", err)
		}

		printPairingInstructions(code, phone)
	} else {
		// Já autenticado - apenas conectar
		if err := client.Connect(); err != nil {
			return false, fmt.Errorf("connect: %v", err)
		}
	}

	return isFirstLogin, nil
}

// printPairingInstructions exibe instruções de pareamento
func printPairingInstructions(code, phone string) {
	log.Println("===========================================")
	log.Printf("CÓDIGO DE PAREAMENTO: %s", code)
	log.Printf("Abra o WhatsApp no telefone %s", phone)
	log.Println("Vá em: Aparelhos conectados -> Conectar um aparelho -> Conectar com número de telefone")
	log.Println("Digite o código acima")
	log.Println("===========================================")
}
