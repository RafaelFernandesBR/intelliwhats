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
	State       string `json:"state"` // PROCESSING, ACTIVE ou FAILED (relevante para vídeo)
}

// GeminiSystemInstruction contém a instrução do sistema
type GeminiSystemInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiTranscriptionRequest representa a requisição de transcrição
type GeminiTranscriptionRequest struct {
	SystemInstruction *GeminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []GeminiContent         `json:"contents"`
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
	SystemInstruction *GeminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []GeminiImageContent    `json:"contents"`
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

// ---- TTS (Text-to-Speech) ----

// GeminiTTSRequest representa a requisição de síntese de voz
type GeminiTTSRequest struct {
	Contents         []GeminiContent    `json:"contents"`
	GenerationConfig GeminiTTSGenConfig `json:"generationConfig"`
}

// GeminiTTSGenConfig configura a saída de áudio e a voz
type GeminiTTSGenConfig struct {
	ResponseModalities []string           `json:"responseModalities"`
	SpeechConfig       GeminiSpeechConfig `json:"speechConfig"`
}

// GeminiSpeechConfig contém a configuração de voz
type GeminiSpeechConfig struct {
	VoiceConfig GeminiVoiceConfig `json:"voiceConfig"`
}

// GeminiVoiceConfig seleciona a voz pré-definida
type GeminiVoiceConfig struct {
	PrebuiltVoiceConfig GeminiPrebuiltVoice `json:"prebuiltVoiceConfig"`
}

// GeminiPrebuiltVoice identifica a voz por nome
type GeminiPrebuiltVoice struct {
	VoiceName string `json:"voiceName"`
}

// GeminiTTSResponse representa a resposta da síntese de voz
type GeminiTTSResponse struct {
	Candidates []GeminiTTSCandidate `json:"candidates"`
}

// GeminiTTSCandidate representa um candidato de resposta de áudio
type GeminiTTSCandidate struct {
	Content GeminiTTSContent `json:"content"`
}

// GeminiTTSContent contém as partes de áudio da resposta
type GeminiTTSContent struct {
	Parts []GeminiTTSPart `json:"parts"`
}

// GeminiTTSPart carrega o áudio inline (base64 PCM)
type GeminiTTSPart struct {
	InlineData *GeminiTTSInlineData `json:"inlineData,omitempty"`
}

// GeminiTTSInlineData contém o áudio PCM em base64 (ex.: audio/L16;rate=24000)
type GeminiTTSInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// ---- Audiodescrição estruturada de vídeo (Fase 2) ----

// ADSegment é um trecho de narração posicionado no tempo do vídeo (numa pausa).
type ADSegment struct {
	StartMs int    `json:"start_ms"`
	Text    string `json:"text"`
}

// GeminiVideoADRequest solicita a audiodescrição em segmentos (JSON estruturado).
type GeminiVideoADRequest struct {
	SystemInstruction *GeminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []GeminiContent          `json:"contents"`
	GenerationConfig  GeminiADGenConfig         `json:"generationConfig"`
}

// GeminiADGenConfig força a saída em JSON conforme um schema.
type GeminiADGenConfig struct {
	ResponseMIMEType string        `json:"responseMimeType"`
	ResponseSchema   *GeminiSchema `json:"responseSchema,omitempty"`
}

// GeminiSchema descreve o formato de resposta esperado (subconjunto OpenAPI).
type GeminiSchema struct {
	Type       string                   `json:"type"`
	Items      *GeminiSchema            `json:"items,omitempty"`
	Properties map[string]*GeminiSchema `json:"properties,omitempty"`
	Required   []string                 `json:"required,omitempty"`
}
