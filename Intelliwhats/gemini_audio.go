package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"intelliwhats/models"
)

// transcribeAudioWithGemini transcreve um áudio usando a Gemini API
func transcribeAudioWithGemini(audioData []byte, mimetype string) (string, error) {
	// Salvar arquivo temporário
	tempFilePath, err := saveAudioToTempFile(audioData, mimetype)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFilePath)

	log.Println("Iniciando upload do áudio para Gemini...")

	// Fazer upload do áudio
	fileURI, err := uploadAudioToGemini(audioData, mimetype)
	if err != nil {
		return "", err
	}

	log.Printf("Arquivo enviado, URI: %s", fileURI)
	log.Println("Solicitando transcrição...")

	// Solicitar transcrição
	transcription, err := requestGeminiTranscription(fileURI, mimetype)
	if err != nil {
		return "", err
	}

	return transcription, nil
}

// saveAudioToTempFile salva o áudio em um arquivo temporário
func saveAudioToTempFile(audioData []byte, mimetype string) (string, error) {
	extension := getAudioExtension(mimetype)
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("audio_%d.%s", time.Now().UnixNano(), extension))

	if err := os.WriteFile(tempFilePath, audioData, 0644); err != nil {
		return "", fmt.Errorf("salvar arquivo temp: %v", err)
	}

	return tempFilePath, nil
}

// uploadAudioToGemini faz upload do áudio para a Gemini API
func uploadAudioToGemini(audioData []byte, mimetype string) (string, error) {
	// Inicializar upload resumable
	uploadURL, err := initializeGeminiUpload(len(audioData), mimetype)
	if err != nil {
		return "", fmt.Errorf("inicializar upload: %v", err)
	}

	log.Println("Upload URL obtida")

	// Fazer upload do arquivo
	fileURI, err := uploadFileToGemini(uploadURL, audioData)
	if err != nil {
		return "", fmt.Errorf("upload arquivo: %v", err)
	}

	return fileURI, nil
}

// initializeGeminiUpload inicializa o upload resumable na Gemini API
func initializeGeminiUpload(fileSize int, mimetype string) (string, error) {
	requestBody := models.GeminiUploadRequest{
		File: models.GeminiFileInfo{
			DisplayName: fmt.Sprintf("audio-%d", time.Now().Unix()),
		},
	}

	jsonData, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", GEMINI_UPLOAD_URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-goog-api-key", GEMINI_API_KEY)
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", fmt.Sprintf("%d", fileSize))
	req.Header.Set("X-Goog-Upload-Header-Content-Type", mimetype)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	uploadURL := resp.Header.Get("x-goog-upload-url")
	if uploadURL == "" {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("URL de upload não obtida. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	return uploadURL, nil
}

// uploadFileToGemini faz o upload do arquivo para a URL obtida
func uploadFileToGemini(uploadURL string, data []byte) (string, error) {
	req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("X-Goog-Upload-Offset", "0")
	req.Header.Set("X-Goog-Upload-Command", "upload, finalize")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload falhou. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	var result models.GeminiUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.File.URI == "" {
		return "", fmt.Errorf("resposta inválida: URI vazia")
	}

	return result.File.URI, nil
}

// requestGeminiTranscription solicita a transcrição do áudio
func requestGeminiTranscription(fileURI, mimetype string) (string, error) {
	requestBody := buildTranscriptionRequest(fileURI, mimetype)

	jsonData, _ := json.Marshal(requestBody)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", GEMINI_MODEL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-goog-api-key", GEMINI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var result models.GeminiTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Extrair texto da transcrição
	transcription, err := extractTranscriptionText(result)
	if err != nil {
		return "", err
	}

	return transcription, nil
}

// buildTranscriptionRequest monta o corpo da requisição de transcrição
func buildTranscriptionRequest(fileURI, mimetype string) models.GeminiTranscriptionRequest {
	return models.GeminiTranscriptionRequest{
		Contents: []models.GeminiContent{
			{
				Parts: []models.GeminiPart{
					{
						Text: getTranscriptionPrompt(),
					},
					{
						FileData: &models.GeminiFileData{
							MimeType: mimetype,
							FileURI:  fileURI,
						},
					},
				},
			},
		},
	}
}

// getTranscriptionPrompt retorna o prompt de transcrição
func getTranscriptionPrompt() string {
	return `Transcreva o áudio em português brasileiro de forma natural e fluida. 
                        
Regras importantes:
- Remova hesitações, repetições desnecessárias e vícios de linguagem (eh, hnn, é, tipo, né quando usado apenas como vícios)
- Corrija erros de fala mantendo o significado original
- Organize o texto de forma clara e coesa
- Preserve o conteúdo e a intenção do que foi dito
- NÃO inclua timestamps, minutagem ou marcações de tempo
- Retorne apenas o texto transcrito de forma natural

Forneça uma transcrição limpa e legível, como se fosse um texto escrito.`
}

// extractTranscriptionText extrai o texto da transcrição da resposta
func extractTranscriptionText(response models.GeminiTranscriptionResponse) (string, error) {
	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("resposta sem candidatos")
	}

	firstCandidate := response.Candidates[0]
	if len(firstCandidate.Content.Parts) == 0 {
		return "", fmt.Errorf("sem parts no conteúdo")
	}

	for _, part := range firstCandidate.Content.Parts {
		if part.Text != "" {
			return part.Text, nil
		}
	}

	return "", fmt.Errorf("texto não encontrado na resposta")
}
