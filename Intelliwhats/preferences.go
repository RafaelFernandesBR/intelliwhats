package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// APIProvider representa o provedor de API para processamento de imagens
type APIProvider int

const (
	APIGrok   APIProvider = 1 // API Grok (padrão)
	APIGemini APIProvider = 2 // API Gemini
)

// initPreferencesTable cria a tabela de preferências se não existir
func initPreferencesTable(ctx context.Context, db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS user_preferences (
		chat_jid TEXT PRIMARY KEY,
		api_provider INTEGER NOT NULL DEFAULT 1,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("criar tabela user_preferences: %v", err)
	}

	log.Println("Tabela user_preferences inicializada")
	return nil
}

// getAPIPreference retorna a API preferida para um chat específico
func getAPIPreference(ctx context.Context, db *sql.DB, chatJID string) (APIProvider, error) {
	var apiProvider int
	query := "SELECT api_provider FROM user_preferences WHERE chat_jid = ?"

	err := db.QueryRowContext(ctx, query, chatJID).Scan(&apiProvider)
	if err == sql.ErrNoRows {
		// Preferência não existe, retornar padrão (Grok)
		return APIGrok, nil
	}
	if err != nil {
		return APIGrok, fmt.Errorf("consultar preferência: %v", err)
	}

	return APIProvider(apiProvider), nil
}

// setAPIPreference define a API preferida para um chat específico
func setAPIPreference(ctx context.Context, db *sql.DB, chatJID string, apiProvider APIProvider) error {
	query := `
	INSERT INTO user_preferences (chat_jid, api_provider, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(chat_jid) DO UPDATE SET
		api_provider = excluded.api_provider,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.ExecContext(ctx, query, chatJID, int(apiProvider))
	if err != nil {
		return fmt.Errorf("salvar preferência: %v", err)
	}

	log.Printf("API %d configurada para %s", apiProvider, chatJID)
	return nil
}

// getAPIProviderName retorna o nome legível da API
func getAPIProviderName(apiProvider APIProvider) string {
	switch apiProvider {
	case APIGrok:
		return "Grok"
	case APIGemini:
		return "Gemini"
	default:
		return "Desconhecida"
	}
}
