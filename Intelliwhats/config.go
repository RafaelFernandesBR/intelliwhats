package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

const (
	GROK_API_URL      = "https://api.x.ai/v1/responses"
	GROK_MODEL        = "grok-4-0709"
	GEMINI_MODEL      = "gemini-2.5-flash"
	GEMINI_UPLOAD_URL = "https://generativelanguage.googleapis.com/upload/v1beta/files"

	// Configurações gerais
	DefaultDBPath  = "./login/whatsmeow.db"
	DefaultTempDir = "./temp"
	HTTPTimeout    = 300 * time.Second
)

var (
	// API Keys (carregadas de variáveis de ambiente)
	GROK_API_KEY   string
	GEMINI_API_KEY string

	// Cliente WhatsApp global
	client *whatsmeow.Client

	// Container do banco de dados (para acessar preferências)
	dbContainer *sqlstore.Container
	db          *sql.DB

	// Cliente HTTP reutilizável
	httpClient = &http.Client{Timeout: HTTPTimeout}

	// Diretório temporário para arquivos
	tempDir = DefaultTempDir
)

// init carrega configurações de variáveis de ambiente
func init() {
	// Carregar API keys
	GROK_API_KEY = os.Getenv("GROK_API_KEY")
	if GROK_API_KEY == "" {
		log.Fatal("GROK_API_KEY não definida. Configure a variável de ambiente.")
	}

	GEMINI_API_KEY = os.Getenv("GEMINI_API_KEY")
	if GEMINI_API_KEY == "" {
		log.Fatal("GEMINI_API_KEY não definida. Configure a variável de ambiente.")
	}
}
