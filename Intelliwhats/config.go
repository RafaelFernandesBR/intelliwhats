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

	// Idioma padrão
	DefaultLanguage = "pt-BR"
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

// GetImageDescriptionPrompt retorna o prompt de descrição de imagem no idioma especificado
func GetImageDescriptionPrompt(language string) string {
	return `Describe the image objectively in ` + language + `, providing a clear overview of visible elements for visually impaired users. Follow these rules strictly:

- Start with general scene (who/what/where).
- Highlight main action and key elements.
- Transcribe all visible text verbatim.
- Use present tense and active verbs.
- Focus on relevant visual information only.
- No introductions, opinions, 'image of', emojis, or redundant phrases.
- Answer only in ` + language + `, pure description.

Describe following this exact structure.`
}

// GetAudioTranscriptionPrompt retorna o prompt de transcrição de áudio no idioma especificado
func GetAudioTranscriptionPrompt(language string) string {
	return `Transcreva o áudio em ` + language + ` de forma natural e fluida. 
                        
Regras importantes:
- Remova hesitações, repetições desnecessárias e vícios de linguagem (eh, hnn, é, tipo, né quando usado apenas como vícios)
- Corrija erros de fala mantendo o significado original
- Organize o texto de forma clara e coesa
- Preserve o conteúdo e a intenção do que foi dito
- NÃO inclua timestamps, minutagem ou marcações de tempo
- Retorne apenas o texto transcrito de forma natural

Forneça uma transcrição limpa e legível, como se fosse um texto escrito.`
}
