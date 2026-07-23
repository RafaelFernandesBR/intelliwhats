package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"intelliwhats/models"
)

const (
	// ttsMinInterval mantém as chamadas ao TTS abaixo do limite da cota
	// (10 req/min no plano grátis) → ~8,5 req/min.
	ttsMinInterval = 7 * time.Second
	// maxTTSAttempts é o número de tentativas por segmento em caso de 429.
	maxTTSAttempts = 4
	// ttsDefaultRetry é usado quando a API não informa um retryDelay.
	ttsDefaultRetry = 30 * time.Second
)

var (
	ttsMu       sync.Mutex
	ttsLastCall time.Time
)

// ttsThrottle espaça as chamadas ao TTS globalmente (entre vídeos/goroutines)
// para respeitar a cota da API.
func ttsThrottle() {
	ttsMu.Lock()
	defer ttsMu.Unlock()
	if wait := ttsMinInterval - time.Since(ttsLastCall); wait > 0 {
		time.Sleep(wait)
	}
	ttsLastCall = time.Now()
}

// synthesizeSpeech converte texto em áudio falado usando o Gemini TTS.
// Aplica pacing entre chamadas e reencaixa em 429 respeitando o retryDelay da API.
// Retorna PCM bruto (16-bit little-endian, 24kHz, mono).
func synthesizeSpeech(text string) ([]byte, error) {
	requestBody := models.GeminiTTSRequest{
		Contents: []models.GeminiContent{
			{Parts: []models.GeminiPart{{Text: text}}},
		},
		GenerationConfig: models.GeminiTTSGenConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: models.GeminiSpeechConfig{
				VoiceConfig: models.GeminiVoiceConfig{
					PrebuiltVoiceConfig: models.GeminiPrebuiltVoice{VoiceName: GEMINI_TTS_VOICE},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request TTS: %v", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", GEMINI_TTS_MODEL)

	var lastErr error
	for attempt := 0; attempt < maxTTSAttempts; attempt++ {
		ttsThrottle()

		pcm, retryAfter, err := doTTSRequest(url, jsonData)
		if err == nil {
			return pcm, nil
		}
		lastErr = err

		// retryAfter > 0 sinaliza um 429 que deve ser repetido; caso contrário, falha imediata
		if retryAfter <= 0 {
			return nil, err
		}
		log.Printf("TTS 429 (cota): aguardando %s (tentativa %d/%d)", retryAfter, attempt+1, maxTTSAttempts)
		time.Sleep(retryAfter)
	}

	return nil, fmt.Errorf("TTS falhou após %d tentativas: %v", maxTTSAttempts, lastErr)
}

// doTTSRequest faz uma requisição ao TTS. Em caso de 429, retorna retryAfter > 0
// com o tempo de espera indicado pela API (ou um padrão).
func doTTSRequest(url string, jsonData []byte) ([]byte, time.Duration, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("x-goog-api-key", GEMINI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("requisição TTS falhou: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Erro de leitura (rede) é transitório → retentável com espera curta
		return nil, 5 * time.Second, fmt.Errorf("ler resposta TTS: %v", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, parseRetryDelay(body), fmt.Errorf("TTS 429 (cota excedida)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("API TTS retornou status %d: %s", resp.StatusCode, string(body))
	}

	var result models.GeminiTTSResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("decodificar resposta TTS: %v", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, 0, fmt.Errorf("resposta TTS sem áudio")
	}

	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != "" {
			pcm, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				return nil, 0, fmt.Errorf("decodificar áudio base64: %v", err)
			}
			return pcm, 0, nil
		}
	}

	return nil, 0, fmt.Errorf("áudio não encontrado na resposta TTS")
}

// parseRetryDelay extrai o retryDelay do corpo de erro 429 do Gemini
// (campo error.details[].retryDelay, ex.: "30s"), com um padrão de fallback.
func parseRetryDelay(body []byte) time.Duration {
	var apiErr struct {
		Error struct {
			Details []struct {
				RetryDelay string `json:"retryDelay"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return ttsDefaultRetry
	}

	for _, d := range apiErr.Error.Details {
		if d.RetryDelay == "" {
			continue
		}
		if dur, err := time.ParseDuration(d.RetryDelay); err == nil && dur > 0 {
			return dur + time.Second // pequena margem de segurança
		}
	}

	return ttsDefaultRetry
}
