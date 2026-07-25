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
	GROK_MODEL        = "grok-4-1-fast-reasoning"
	GEMINI_MODEL      = "gemini-3.1-flash-lite"
	GEMINI_UPLOAD_URL = "https://generativelanguage.googleapis.com/upload/v1beta/files"

	// Text-to-Speech (audiodescrição de vídeo)
	GEMINI_TTS_MODEL = "gemini-2.5-flash-preview-tts"
	GEMINI_TTS_VOICE = "Kore"

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

// GetVideoAudioDescriptionPrompt retorna o prompt de audiodescrição de vídeo no idioma especificado.
// Gera um texto de narração único descrevendo a cena visual e o que é falado, pronto para TTS.
func GetVideoAudioDescriptionPrompt(language string) string {
	return `Você é um audiodescritor profissional. Assista ao vídeo e produza uma narração em ` + language + ` para pessoas com deficiência visual.

Regras:
- Descreva objetivamente os elementos visuais principais: quem/o quê/onde, ação principal e mudanças de cena.
- Integre naturalmente o que é dito no áudio (falas relevantes), sem inventar.
- Use presente do indicativo e verbos ativos.
- Transcreva textos visíveis importantes na tela.
- Sem introduções, opiniões, "vídeo de", emojis ou frases redundantes.
- Retorne apenas o texto da narração corrida, pronto para ser lido em voz alta em ` + language + `.`
}

// GetVideoADSegmentsPrompt retorna o prompt de audiodescrição segmentada de vídeo.
// Pede segmentos curtos posicionados nas pausas do diálogo, com o instante de início.
func GetVideoADSegmentsPrompt(language string) string {
	return `Você é um audiodescritor profissional. Assista ao vídeo e produza uma audiodescrição em ` + language + ` para pessoas com deficiência visual, dividida em segmentos curtos.

Regras:
- COBERTURA TOTAL: o vídeo pode ter vários minutos. Distribua a audiodescrição do começo até o FIM, cobrindo toda a duração. É OBRIGATÓRIO descrever também o meio e o final — NUNCA concentre tudo nos primeiros segundos.
- Ancore em EVENTOS: descreva mudanças de cena e mudanças visuais relevantes (nova pessoa ou objeto, ação importante, mudança de local, texto na tela). Mas NÃO deixe trechos longos sem nenhuma narração: se a cena ficar parada por muito tempo, descreva periodicamente o que está em quadro.
- ESPAÇAMENTO: cada narração falada leva alguns segundos para ser dita, então deixe pelo menos ~5 segundos entre o início (start_ms) de um segmento e o próximo. NUNCA agrupe vários segmentos nos mesmos poucos segundos — se houver muitos eventos juntos, escolha apenas o mais importante.
- Cada segmento deve ser BREVE: uma única frase curta, no máximo ~12 palavras, descrevendo o essencial daquele momento. Vá direto ao ponto.
- Informe o instante de início de cada segmento em milissegundos (start_ms). Quando houver pausas na fala, prefira encaixar a narração nelas; mas pode narrar durante os diálogos quando o evento for importante (o áudio original é abaixado automaticamente enquanto a narração toca).
- Não repita literalmente o que é dito no áudio; complemente com a informação visual.
- Presente do indicativo, verbos ativos; sem introduções, opiniões, "vídeo de", emojis ou frases redundantes.
- Responda apenas com o JSON no formato especificado, em ` + language + `.`
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
