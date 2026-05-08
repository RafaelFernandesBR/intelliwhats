package models

// GeminiUploadRequest representa a requisição para inicializar o upload
type GeminiUploadRequest struct {
	File GeminiFileInfo `json:"file"`
}

// GeminiFileInfo contém informações do arquivo para upload
type GeminiFileInfo struct {
	DisplayName string `json:"display_name"`
}

// GeminiUploadResponse representa a resposta do upload
type GeminiUploadResponse struct {
	File GeminiFile `json:"file"`
}

// GeminiFile contém informações do arquivo após upload
type GeminiFile struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	MimeType    string `json:"mimeType"`
	SizeBytes   string `json:"sizeBytes"`
	URI         string `json:"uri"`
}

// GeminiTranscriptionRequest representa a requisição de transcrição
type GeminiTranscriptionRequest struct {
	Contents []GeminiContent `json:"contents"`
}

// GeminiContent representa o conteúdo da requisição
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart pode ser texto ou arquivo
type GeminiPart struct {
	Text     string          `json:"text,omitempty"`
	FileData *GeminiFileData `json:"file_data,omitempty"`
}

// GeminiFileData contém dados do arquivo para transcrição
type GeminiFileData struct {
	MimeType string `json:"mime_type"`
	FileURI  string `json:"file_uri"`
}

// GeminiTranscriptionResponse representa a resposta da transcrição
type GeminiTranscriptionResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// GeminiCandidate representa um candidato de resposta
type GeminiCandidate struct {
	Content GeminiResponseContent `json:"content"`
}

// GeminiResponseContent contém o conteúdo da resposta
type GeminiResponseContent struct {
	Parts []GeminiResponsePart `json:"parts"`
}

// GeminiResponsePart representa uma parte da resposta
type GeminiResponsePart struct {
	Text string `json:"text"`
}

// GeminiImageRequest representa a requisição de descrição de imagem
type GeminiImageRequest struct {
	Contents []GeminiImageContent `json:"contents"`
}

// GeminiImageContent representa o conteúdo da requisição de imagem
type GeminiImageContent struct {
	Role  string            `json:"role"`
	Parts []GeminiImagePart `json:"parts"`
}

// GeminiImagePart pode ser texto ou imagem inline
type GeminiImagePart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`
}

// GeminiInlineData contém dados de imagem em base64
type GeminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

// GeminiImageResponse representa a resposta da descrição de imagem
type GeminiImageResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}
