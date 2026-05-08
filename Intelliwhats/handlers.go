package main

import (
	"context"
	"fmt"
	"log"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// eventHandler processa todos os eventos do WhatsApp
func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Ignorar mensagens próprias
		if v.Info.IsFromMe {
			return
		}
		go handleMessage(v)

	case *events.Connected:
		log.Println("Conectado ao WhatsApp")

	case *events.LoggedOut:
		log.Printf("Desconectado: %v", v.Reason)
	}
}

// handleMessage processa mensagens recebidas
func handleMessage(msg *events.Message) {
	ctx := context.Background()

	// Verificar tipo de mensagem
	if msg.Message.GetImageMessage() != nil {
		handleImageMessage(ctx, msg)
	} else if msg.Message.GetStickerMessage() != nil {
		handleStickerMessage(ctx, msg)
	} else if msg.Message.GetAudioMessage() != nil {
		handleAudioMessage(ctx, msg)
	} else {
		handleTextMessage(ctx, msg)
	}
}

// handleImageMessage processa mensagens de imagem
func handleImageMessage(ctx context.Context, msg *events.Message) {
	log.Printf("Recebendo imagem de %s", msg.Info.Sender.String())

	// Baixar imagem
	imageData, err := client.Download(ctx, msg.Message.GetImageMessage())
	if err != nil {
		log.Printf("Erro ao baixar imagem: %v", err)
		replyToMessage(ctx, msg, "Erro ao baixar a imagem.")
		return
	}

	// Verificar qual API usar
	chatJID := msg.Info.Chat.String()
	apiProvider, err := getAPIPreference(ctx, db, chatJID)
	if err != nil {
		log.Printf("Erro ao obter preferência de API: %v", err)
		apiProvider = APIGrok // Fallback para Grok
	}

	// Processar com a API selecionada
	var description string
	switch apiProvider {
	case APIGemini:
		description, err = processImageWithGemini(imageData)
	default: // APIGrok
		description, err = processImageWithGrok(imageData)
	}

	if err != nil {
		log.Printf("Erro ao processar imagem: %v", err)
		replyToMessage(ctx, msg, "Não foi possível reconhecer o conteúdo da imagem.")
		return
	}

	log.Printf("Imagem processada com sucesso usando %s", getAPIProviderName(apiProvider))
	replyToMessage(ctx, msg, description)
}

// handleStickerMessage processa mensagens de figurinha (sticker)
func handleStickerMessage(ctx context.Context, msg *events.Message) {
	log.Printf("Recebendo figurinha de %s", msg.Info.Sender.String())

	// Baixar figurinha
	stickerData, err := client.Download(ctx, msg.Message.GetStickerMessage())
	if err != nil {
		log.Printf("Erro ao baixar figurinha: %v", err)
		replyToMessage(ctx, msg, "Erro ao baixar a figurinha.")
		return
	}

	// Verificar qual API usar
	chatJID := msg.Info.Chat.String()
	apiProvider, err := getAPIPreference(ctx, db, chatJID)
	if err != nil {
		log.Printf("Erro ao obter preferência de API: %v", err)
		apiProvider = APIGrok // Fallback para Grok
	}

	// Processar com a API selecionada
	var description string
	switch apiProvider {
	case APIGemini:
		description, err = processImageWithGemini(stickerData)
	default: // APIGrok
		description, err = processImageWithGrok(stickerData)
	}

	if err != nil {
		log.Printf("Erro ao processar figurinha: %v", err)
		replyToMessage(ctx, msg, "Não foi possível reconhecer o conteúdo da figurinha.")
		return
	}

	log.Printf("Figurinha processada com sucesso usando %s", getAPIProviderName(apiProvider))
	replyToMessage(ctx, msg, description)
}

// handleAudioMessage processa mensagens de áudio
func handleAudioMessage(ctx context.Context, msg *events.Message) {
	log.Printf("Recebendo áudio de %s", msg.Info.Sender.String())

	audioMsg := msg.Message.GetAudioMessage()

	// Baixar áudio
	audioData, err := client.Download(ctx, audioMsg)
	if err != nil {
		log.Printf("Erro ao baixar áudio: %v", err)
		replyToMessage(ctx, msg, "Erro ao baixar o áudio.")
		return
	}

	// Determinar mimetype
	mimetype := audioMsg.GetMimetype()
	if mimetype == "" {
		mimetype = "audio/ogg; codecs=opus"
	}

	log.Printf("Áudio recebido com mimetype: %s", mimetype)

	// Transcrever com Gemini API
	transcription, err := transcribeAudioWithGemini(audioData, mimetype)
	if err != nil {
		log.Printf("Erro ao transcrever áudio: %v", err)
		replyToMessage(ctx, msg, "Erro ao processar o áudio. Tente novamente.")
		return
	}

	log.Println("Transcrição concluída com sucesso")
	replyToMessage(ctx, msg, fmt.Sprintf("📝 Transcrição:\n\n%s", transcription))
}

// handleTextMessage processa mensagens de texto
func handleTextMessage(ctx context.Context, msg *events.Message) {
	text := extractText(msg.Message)

	// Comando ping/pong
	if text == "!ping" {
		replyToMessage(ctx, msg, "pong")
		return
	}

	// Comando para trocar API de processamento de imagens
	if len(text) >= 5 && text[:4] == "!api" {
		handleAPICommand(ctx, msg, text)
		return
	}
}

// handleAPICommand processa o comando !api para trocar a API de processamento de imagens
func handleAPICommand(ctx context.Context, msg *events.Message, text string) {
	chatJID := msg.Info.Chat.String()

	// Parsear comando (!api 1 ou !api 2)
	var apiProvider APIProvider
	switch text {
	case "!api 1":
		apiProvider = APIGrok
	case "!api 2":
		apiProvider = APIGemini
	case "!api":
		// Mostrar API atual
		currentAPI, err := getAPIPreference(ctx, db, chatJID)
		if err != nil {
			log.Printf("Erro ao obter API atual: %v", err)
			replyToMessage(ctx, msg, "Erro ao consultar configuração atual.")
			return
		}
		replyToMessage(ctx, msg, fmt.Sprintf("🤖 API atual: *%s*\n\nUse:\n• !api 1 - Grok\n• !api 2 - Gemini", getAPIProviderName(currentAPI)))
		return
	default:
		replyToMessage(ctx, msg, "❌ Comando inválido.\n\nUse:\n• !api - Ver API atual\n• !api 1 - Usar Grok\n• !api 2 - Usar Gemini")
		return
	}

	// Salvar preferência no banco
	err := setAPIPreference(ctx, db, chatJID, apiProvider)
	if err != nil {
		log.Printf("Erro ao salvar preferência de API: %v", err)
		replyToMessage(ctx, msg, "Erro ao salvar configuração.")
		return
	}

	replyToMessage(ctx, msg, fmt.Sprintf("✅ API alterada para *%s*\n\nAgora todas as imagens e figurinhas serão processadas com esta API.", getAPIProviderName(apiProvider)))
}

// replyToMessage envia uma resposta para uma mensagem
func replyToMessage(ctx context.Context, msg *events.Message, text string) {
	_, err := client.SendMessage(ctx, msg.Info.Chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		log.Printf("Erro ao enviar resposta: %v", err)
	}
}
