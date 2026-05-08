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

func processImageWithGrok(imageData []byte) (string, error) {
	// Converter para base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Idioma parametrizado
	language := "pt-BR"

	// Criar requisição
	requestBody := buildGrokRequest(base64Image, GetImageDescriptionPrompt(language))

	// Enviar para API
	response, err := sendGrokRequest(requestBody)
	if err != nil {
		return "", err
	}

	// Extrair descrição da resposta
	description, err := extractGrokDescription(response)
	if err != nil {
		return "", err
	}

	return description, nil
}

func buildGrokRequest(base64Image, promptText string) models.GrokRequest {
	return models.GrokRequest{
		Input: []models.GrokInput{
			{
				Role: "user",
				Content: []models.GrokContent{
					{
						Type:     "input_image",
						ImageURL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						Detail:   "high",
					},
					{
						Type: "input_text",
						Text: promptText,
					},
				},
			},
		},
		Model: GROK_MODEL,
	}
}

func sendGrokRequest(requestBody models.GrokRequest) (models.GrokResponse, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return models.GrokResponse{}, fmt.Errorf("marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", GROK_API_URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return models.GrokResponse{}, fmt.Errorf("criar request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+GROK_API_KEY)

	resp, err := httpClient.Do(req)
	if err != nil {
		return models.GrokResponse{}, fmt.Errorf("requisição falhou: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return models.GrokResponse{}, fmt.Errorf("API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var result models.GrokResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.GrokResponse{}, fmt.Errorf("decodificar resposta: %v", err)
	}

	return result, nil
}

// extractGrokDescription extrai o texto de descrição da resposta da Grok API
func extractGrokDescription(response models.GrokResponse) (string, error) {
	if len(response.Output) == 0 {
		return "", fmt.Errorf("resposta inválida: sem output")
	}

	firstOutput := response.Output[0]
	if len(firstOutput.Content) == 0 {
		return "", fmt.Errorf("resposta inválida: sem content")
	}

	for _, item := range firstOutput.Content {
		if item.Type == "output_text" && item.Text != "" {
			return item.Text, nil
		}
	}

	return "", fmt.Errorf("texto não encontrado na resposta")
}
