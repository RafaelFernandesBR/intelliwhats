package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"intelliwhats/models"
)

// analyzeVideoWithGemini faz upload do vídeo, aguarda o processamento e retorna
// os segmentos de audiodescrição posicionados no tempo (nas pausas do diálogo).
// Se a saída estruturada falhar (R1), recorre a uma narração única no início.
func analyzeVideoWithGemini(videoData []byte, mimetype string) ([]models.ADSegment, error) {
	log.Println("Iniciando upload do vídeo para Gemini...")

	file, err := uploadVideoToGemini(videoData, mimetype)
	if err != nil {
		return nil, err
	}
	// Remove o arquivo do Gemini após o uso (não armazenar mídia em terceiros além do necessário)
	defer deleteGeminiFile(file.Name)
	log.Printf("Vídeo enviado, URI: %s (aguardando processamento)", file.URI)

	if err := waitForGeminiFileActive(file.Name); err != nil {
		return nil, err
	}

	log.Println("Vídeo ativo, solicitando audiodescrição segmentada...")
	segments, err := requestGeminiADSegments(file.URI, mimetype)
	if err == nil {
		segments = sanitizeSegments(segments)
		if len(segments) > 0 {
			log.Printf("Audiodescrição com %d segmento(s)", len(segments))
			return segments, nil
		}
	}

	// Fallback (R1): timestamps/JSON indisponíveis → narração única no início
	if err != nil {
		log.Printf("Segmentação falhou (%v); usando narração única como fallback", err)
	} else {
		log.Println("Nenhum segmento válido retornado; usando narração única como fallback")
	}

	description, ferr := requestGeminiVideoDescription(file.URI, mimetype)
	if ferr != nil {
		return nil, ferr
	}
	return []models.ADSegment{{StartMs: 0, Text: description}}, nil
}

// maxADSegments limita o número de segmentos processados (custo/latência de TTS).
const maxADSegments = 40

// sanitizeSegments descarta segmentos inválidos, ordena por tempo de início e
// limita a quantidade para conter custo/latência.
func sanitizeSegments(segments []models.ADSegment) []models.ADSegment {
	cleaned := make([]models.ADSegment, 0, len(segments))
	for _, s := range segments {
		if s.StartMs < 0 || strings.TrimSpace(s.Text) == "" {
			continue
		}
		cleaned = append(cleaned, s)
	}
	sort.SliceStable(cleaned, func(i, j int) bool { return cleaned[i].StartMs < cleaned[j].StartMs })
	if len(cleaned) > maxADSegments {
		log.Printf("Aviso: %d segmentos retornados; limitando a %d", len(cleaned), maxADSegments)
		cleaned = cleaned[:maxADSegments]
	}
	return cleaned
}

// requestGeminiADSegments solicita a audiodescrição em segmentos (JSON estruturado).
func requestGeminiADSegments(fileURI, mimetype string) ([]models.ADSegment, error) {
	requestBody := models.GeminiVideoADRequest{
		SystemInstruction: &models.GeminiSystemInstruction{
			Parts: []models.GeminiPart{
				{Text: GetVideoADSegmentsPrompt(DefaultLanguage)},
			},
		},
		Contents: []models.GeminiContent{
			{
				Parts: []models.GeminiPart{
					{
						FileData: &models.GeminiFileData{
							MimeType: mimetype,
							FileURI:  fileURI,
						},
					},
				},
			},
		},
		GenerationConfig: models.GeminiADGenConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema: &models.GeminiSchema{
				Type: "ARRAY",
				Items: &models.GeminiSchema{
					Type: "OBJECT",
					Properties: map[string]*models.GeminiSchema{
						"start_ms": {Type: "INTEGER"},
						"text":     {Type: "STRING"},
					},
					Required: []string{"start_ms", "text"},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(requestBody)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", GEMINI_MODEL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-goog-api-key", GEMINI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var result models.GeminiTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// A saída estruturada vem como texto JSON na primeira part
	jsonText, err := extractTranscriptionText(result)
	if err != nil {
		return nil, err
	}

	var segments []models.ADSegment
	if err := json.Unmarshal([]byte(jsonText), &segments); err != nil {
		return nil, fmt.Errorf("parsear segmentos JSON: %v", err)
	}

	return segments, nil
}

// uploadVideoToGemini faz upload do vídeo reutilizando a cadeia resumable do áudio.
func uploadVideoToGemini(videoData []byte, mimetype string) (models.GeminiFile, error) {
	uploadURL, err := initializeGeminiUpload(len(videoData), mimetype, fmt.Sprintf("video-%d", time.Now().Unix()))
	if err != nil {
		return models.GeminiFile{}, fmt.Errorf("inicializar upload: %v", err)
	}

	file, err := uploadFileToGemini(uploadURL, videoData)
	if err != nil {
		return models.GeminiFile{}, fmt.Errorf("upload arquivo: %v", err)
	}

	return file, nil
}

// deleteGeminiFile remove um arquivo do Gemini Files API (best-effort, sem bloquear em erro).
func deleteGeminiFile(fileName string) {
	if fileName == "" {
		return
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s", fileName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("x-goog-api-key", GEMINI_API_KEY)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Aviso: falha ao remover arquivo do Gemini: %v", err)
		return
	}
	resp.Body.Close()
}

// waitForGeminiFileActive aguarda o arquivo sair do estado PROCESSING.
// Vídeos precisam ser processados pelo Gemini antes do generateContent.
func waitForGeminiFileActive(fileName string) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s", fileName)

	// Até ~4 min (120 x 2s): vídeos maiores levam mais tempo para sair de PROCESSING
	const maxAttempts = 120
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("x-goog-api-key", GEMINI_API_KEY)

		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("consultar estado do arquivo: status %d: %s", resp.StatusCode, string(body))
		}

		var file models.GeminiFile
		if err := json.Unmarshal(body, &file); err != nil {
			return fmt.Errorf("decodificar estado do arquivo: %v", err)
		}

		switch file.State {
		case "ACTIVE":
			return nil
		case "FAILED":
			return fmt.Errorf("processamento do vídeo falhou no Gemini")
		default: // PROCESSING ou vazio
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("tempo esgotado aguardando o processamento do vídeo")
}

// requestGeminiVideoDescription solicita a audiodescrição do vídeo já enviado.
func requestGeminiVideoDescription(fileURI, mimetype string) (string, error) {
	requestBody := models.GeminiTranscriptionRequest{
		SystemInstruction: &models.GeminiSystemInstruction{
			Parts: []models.GeminiPart{
				{Text: GetVideoAudioDescriptionPrompt(DefaultLanguage)},
			},
		},
		Contents: []models.GeminiContent{
			{
				Parts: []models.GeminiPart{
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

	// Reutiliza o extrator de texto da transcrição de áudio
	return extractTranscriptionText(result)
}
