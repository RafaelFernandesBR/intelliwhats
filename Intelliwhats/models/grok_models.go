package models

// GrokRequest representa a requisição para a API Grok
type GrokRequest struct {
	Input []GrokInput `json:"input"`
	Model string      `json:"model"`
}

// GrokInput representa uma entrada da requisição
type GrokInput struct {
	Role    string        `json:"role"`
	Content []GrokContent `json:"content"`
}

// GrokContent pode ser imagem ou texto
type GrokContent struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Text     string `json:"text,omitempty"`
}

// GrokResponse representa a resposta da API Grok
type GrokResponse struct {
	Output []GrokOutput `json:"output"`
}

// GrokOutput representa uma saída da resposta
type GrokOutput struct {
	Role    string                `json:"role"`
	Content []GrokResponseContent `json:"content"`
}

// GrokResponseContent representa o conteúdo da resposta
type GrokResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
