package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"go.mau.fi/whatsmeow/types/events"

	"intelliwhats/models"
)

// producedVideo agrupa o vídeo final com audiodescrição e seus metadados.
type producedVideo struct {
	Data      []byte
	Seconds   uint32
	Width     uint32
	Height    uint32
	Thumbnail []byte
}

// handleVideoMessage processa mensagens de vídeo gerando uma audiodescrição
// integrada à faixa de áudio do próprio vídeo. Vídeo sempre usa Gemini
// (Grok não processa vídeo), independentemente da preferência do chat.
func handleVideoMessage(ctx context.Context, msg *events.Message) {
	log.Printf("Recebendo vídeo de %s", msg.Info.Sender.String())

	videoMsg := msg.Message.GetVideoMessage()

	videoData, err := client.Download(ctx, videoMsg)
	if err != nil {
		log.Printf("Erro ao baixar vídeo: %v", err)
		replyToMessage(ctx, msg, "Erro ao baixar o vídeo.")
		return
	}

	mimetype := videoMsg.GetMimetype()
	if mimetype == "" {
		mimetype = "video/mp4"
	}

	replyToMessage(ctx, msg, "🎬 Gerando a audiodescrição do vídeo. Isso pode levar alguns minutos em vídeos longos...")

	// 1. Analisar o vídeo com o Gemini (segmentos de audiodescrição nas pausas)
	segments, err := analyzeVideoWithGemini(videoData, mimetype)
	if err != nil {
		log.Printf("Erro ao analisar vídeo: %v", err)
		replyToMessage(ctx, msg, "Não foi possível gerar a audiodescrição do vídeo.")
		return
	}

	// 2. Sintetizar a narração de cada segmento e integrá-la ao vídeo
	produced, err := buildAudioDescribedVideo(videoData, mimetype, segments)
	if err != nil {
		log.Printf("Erro ao montar vídeo com audiodescrição: %v", err)
		replyToMessage(ctx, msg, "Erro ao montar o vídeo com a audiodescrição.")
		return
	}

	// 3. Enviar o vídeo de volta
	if err := sendVideoReply(ctx, msg, produced); err != nil {
		log.Printf("Erro ao enviar vídeo: %v", err)
		replyToMessage(ctx, msg, "Erro ao enviar o vídeo com a audiodescrição.")
		return
	}

	log.Println("Vídeo com audiodescrição enviado com sucesso")
}

// estimateDurationMs estima a duração da fala pelo tamanho do texto, usado como
// fallback quando a duração real do áudio não pôde ser medida (~70 ms/caractere, mín. 800 ms).
func estimateDurationMs(text string) int {
	ms := len(text) * 70
	if ms < 800 {
		ms = 800
	}
	return ms
}

// buildAudioDescribedVideo sintetiza a narração (TTS) de cada segmento e integra
// os clipes ao vídeo original, posicionados no tempo indicado por cada segmento.
func buildAudioDescribedVideo(videoData []byte, mimetype string, segments []models.ADSegment) (*producedVideo, error) {
	// Salvar o vídeo original em arquivo temporário
	videoExt := getVideoExtension(mimetype)
	videoPath := filepath.Join(tempDir, fmt.Sprintf("video_%d.%s", time.Now().UnixNano(), videoExt))
	if err := os.WriteFile(videoPath, videoData, 0644); err != nil {
		return nil, fmt.Errorf("salvar vídeo temp: %v", err)
	}
	defer os.Remove(videoPath)

	// Duração do vídeo, usada para descartar segmentos posicionados além do fim
	// e para calcular a extensão do vídeo. Se indisponível, ambos são desativados.
	videoDurSec, err := mediaDurationSeconds(videoPath)
	if err != nil {
		log.Printf("Aviso: duração do vídeo indisponível (%v); descarte de segmentos e extensão desativados", err)
	}
	videoDurMs := int(math.Round(videoDurSec * 1000))

	// Sintetizar cada segmento (TTS) e convertê-lo em um clipe WAV posicionado.
	// lastEndMs guarda o fim da última narração aceita, para evitar sobreposição.
	var clips []narrationClip
	lastEndMs := 0
	for i, seg := range segments {
		// Descarta segmentos que começam além da duração do vídeo
		if videoDurMs > 0 && seg.StartMs >= videoDurMs {
			log.Printf("Aviso: segmento %d começa (%d ms) além da duração do vídeo (%d ms); ignorando", i, seg.StartMs, videoDurMs)
			continue
		}
		// Evita sobreposição com a narração anterior (mantém a fala inteligível)
		if seg.StartMs < lastEndMs {
			log.Printf("Aviso: segmento %d (%d ms) sobrepõe a narração anterior (fim em %d ms); ignorando", i, seg.StartMs, lastEndMs)
			continue
		}

		pcm, err := synthesizeSpeech(seg.Text)
		if err != nil {
			log.Printf("Aviso: TTS falhou no segmento %d (%v); ignorando", i, err)
			continue
		}

		wavPath := filepath.Join(tempDir, fmt.Sprintf("narr_%d_%d.wav", time.Now().UnixNano(), i))
		defer os.Remove(wavPath)
		if err := pcmToWavFile(pcm, wavPath); err != nil {
			log.Printf("Aviso: conversão do segmento %d falhou (%v); ignorando", i, err)
			continue
		}

		durMs := estimateDurationMs(seg.Text)
		if d, derr := mediaDurationSeconds(wavPath); derr == nil {
			durMs = int(d * 1000)
		} else {
			log.Printf("Aviso: duração do segmento %d indisponível (%v); usando estimativa", i, derr)
		}

		clips = append(clips, narrationClip{
			wavPath:    wavPath,
			offsetMs:   seg.StartMs,
			durationMs: durMs,
		})
		lastEndMs = seg.StartMs + durMs
	}

	if len(clips) == 0 {
		return nil, fmt.Errorf("nenhuma narração pôde ser gerada")
	}

	// Se a última narração passa do fim do vídeo, estende o vídeo (congela o último
	// frame) em vez de cortar a narração. freezeTailMs dá uma pequena folga no fim;
	// extendToleranceMs evita reencode por diferenças mínimas de arredondamento;
	// maxExtendMs limita a extensão (e o custo de reencode) em casos patológicos.
	const (
		freezeTailMs      = 300
		extendToleranceMs = 50
		maxExtendMs       = 15000
	)
	extendMs := 0
	if videoDurMs > 0 && lastEndMs-videoDurMs > extendToleranceMs {
		extendMs = lastEndMs - videoDurMs + freezeTailMs
		if extendMs > maxExtendMs {
			log.Printf("Aviso: extensão necessária (%d ms) excede o limite; truncando para %d ms", extendMs, maxExtendMs)
			extendMs = maxExtendMs
		}
		log.Printf("Última narração passa do vídeo em %d ms; estendendo (congelando último frame)", lastEndMs-videoDurMs)
	}

	// Integrar as narrações ao vídeo (cada clipe no seu offset, com ducking temporizado)
	outPath := filepath.Join(tempDir, fmt.Sprintf("video_ad_%d.mp4", time.Now().UnixNano()))
	defer os.Remove(outPath)
	if err := muxNarrationIntoVideo(videoPath, clips, extendMs, outPath); err != nil {
		return nil, err
	}

	outData, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("ler vídeo final: %v", err)
	}

	// Metadados (best-effort; falhas não impedem o envio)
	seconds, _ := videoDurationSeconds(outPath)
	width, height, _ := videoDimensions(outPath)
	thumbnail, _ := extractVideoThumbnail(outPath)

	return &producedVideo{
		Data:      outData,
		Seconds:   seconds,
		Width:     width,
		Height:    height,
		Thumbnail: thumbnail,
	}, nil
}
