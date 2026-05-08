package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"intelliwhats/models"
)

// processImageWithGemini processa uma imagem usando a Gemini API
func processImageWithGemini(imageData []byte) (string, error) {
	// Converter para base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Idioma parametrizado
	language := "pt-BR"

	// Criar requisição
	requestBody := buildGeminiImageRequest(base64Image, GetImageDescriptionPrompt(language))

	// Enviar para API
	response, err := sendGeminiImageRequest(requestBody)
	if err != nil {
		return "", err
	}

	// Extrair descrição da resposta
	description, err := extractGeminiImageDescription(response)
	if err != nil {
		return "", err
	}

	return description, nil
}

// buildGeminiImageRequest monta o corpo da requisição de descrição de imagem
func buildGeminiImageRequest(base64Image, promptText string) models.GeminiImageRequest {
	return models.GeminiImageRequest{
		Contents: []models.GeminiImageContent{
			{
				Role: "user",
				Parts: []models.GeminiImagePart{
					{
						InlineData: &models.GeminiInlineData{
							MimeType: "image/png",
							Data:     base64Image,
						},
					},
					{
						Text: promptText,
					},
				},
			},
		},
	}
}

// sendGeminiImageRequest envia a requisição para a Gemini API
func sendGeminiImageRequest(requestBody models.GeminiImageRequest) (models.GeminiImageResponse, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return models.GeminiImageResponse{}, fmt.Errorf("marshal request: %v", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		GEMINI_MODEL, GEMINI_API_KEY)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return models.GeminiImageResponse{}, fmt.Errorf("criar request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return models.GeminiImageResponse{}, fmt.Errorf("requisição falhou: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return models.GeminiImageResponse{}, fmt.Errorf("API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var result models.GeminiImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.GeminiImageResponse{}, fmt.Errorf("decodificar resposta: %v", err)
	}

	return result, nil
}

// extractGeminiImageDescription extrai o texto de descrição da resposta da Gemini API
func extractGeminiImageDescription(response models.GeminiImageResponse) (string, error) {
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
