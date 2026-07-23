package main

import (
	"os"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// getenv retorna o valor de uma variável de ambiente ou um valor padrão
func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// extractText extrai o texto de uma mensagem do WhatsApp
func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	if msg.Conversation != nil {
		return *msg.Conversation
	}

	if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		return *msg.ExtendedTextMessage.Text
	}

	return ""
}

// getAudioExtension determina a extensão do arquivo baseada no mimetype
func getAudioExtension(mimetype string) string {
	if strings.Contains(mimetype, "ogg") {
		return "ogg"
	}
	if strings.Contains(mimetype, "mp4") {
		return "mp4"
	}
	if strings.Contains(mimetype, "mpeg") {
		return "mp3"
	}
	if strings.Contains(mimetype, "wav") {
		return "wav"
	}
	return "mp3"
}

// getVideoExtension determina a extensão do arquivo de vídeo baseada no mimetype
func getVideoExtension(mimetype string) string {
	if strings.Contains(mimetype, "3gpp") {
		return "3gp"
	}
	if strings.Contains(mimetype, "quicktime") {
		return "mov"
	}
	if strings.Contains(mimetype, "webm") {
		return "webm"
	}
	return "mp4"
}
