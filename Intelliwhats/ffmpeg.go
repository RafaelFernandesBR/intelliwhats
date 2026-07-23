package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// duckVolume é o volume aplicado ao áudio original enquanto a narração toca,
// para manter a audiodescrição inteligível (0.0 = mudo, 1.0 = volume cheio).
const duckVolume = 0.35

// ttsSpeedFactor acelera a narração do TTS (2.0 = 2x mais rápido). Preserva o tom
// via atempo; também reduz a duração, o que diminui as sobreposições entre segmentos.
const ttsSpeedFactor = 1.5

// narrationClip representa um trecho de narração a ser inserido no vídeo.
type narrationClip struct {
	wavPath    string // caminho do arquivo WAV com a narração
	offsetMs   int    // deslocamento (ms) em que a narração começa no vídeo
	durationMs int    // duração (ms) da narração, usada para o ducking temporizado
}

// checkFFmpegAvailable verifica se ffmpeg e ffprobe estão disponíveis no PATH.
func checkFFmpegAvailable() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg não encontrado no PATH")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe não encontrado no PATH")
	}
	return nil
}

// runFFmpeg executa o ffmpeg com os argumentos dados, capturando o stderr em caso de erro.
func runFFmpeg(args ...string) error {
	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %v: %s", err, stderr.String())
	}
	return nil
}

// runFFprobe executa o ffprobe e retorna a saída padrão já sem espaços.
func runFFprobe(args ...string) (string, error) {
	cmd := exec.Command("ffprobe", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe: %v: %s", err, stderr.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// pcmToWavFile converte PCM bruto (s16le, 24kHz, mono) vindo do TTS em um arquivo WAV,
// acelerando a narração por ttsSpeedFactor (atempo preserva o tom).
func pcmToWavFile(pcm []byte, outPath string) error {
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "s16le", "-ar", "24000", "-ac", "1",
		"-i", "pipe:0",
		"-filter:a", fmt.Sprintf("atempo=%.2f", ttsSpeedFactor),
		outPath)
	cmd.Stdin = bytes.NewReader(pcm)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pcm->wav: %v: %s", err, stderr.String())
	}
	return nil
}

// videoHasAudio informa se o vídeo possui ao menos uma faixa de áudio.
func videoHasAudio(path string) (bool, error) {
	out, err := runFFprobe("-v", "error", "-select_streams", "a",
		"-show_entries", "stream=index", "-of", "csv=p=0", path)
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// mediaDurationSeconds retorna a duração (em segundos, com fração) de qualquer mídia.
func mediaDurationSeconds(path string) (float64, error) {
	out, err := runFFprobe("-v", "error", "-show_entries", "format=duration",
		"-of", "csv=p=0", path)
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(out, 64)
	if err != nil {
		return 0, fmt.Errorf("duração inválida %q: %v", out, err)
	}
	return f, nil
}

// videoDurationSeconds retorna a duração do vídeo em segundos (arredondada).
func videoDurationSeconds(path string) (uint32, error) {
	f, err := mediaDurationSeconds(path)
	if err != nil {
		return 0, err
	}
	return uint32(math.Round(f)), nil
}

// videoDimensions retorna largura e altura do vídeo.
func videoDimensions(path string) (uint32, uint32, error) {
	out, err := runFFprobe("-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", path)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(out, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("dimensões inválidas: %q", out)
	}
	w, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return uint32(w), uint32(h), nil
}

// extractVideoThumbnail extrai o primeiro frame do vídeo como JPEG (para o preview no WhatsApp).
func extractVideoThumbnail(path string) ([]byte, error) {
	tmp := filepath.Join(tempDir, fmt.Sprintf("thumb_%d.jpg", time.Now().UnixNano()))
	defer os.Remove(tmp)

	if err := runFFmpeg("-y", "-i", path,
		"-frames:v", "1", "-vf", "scale=320:-1", "-q:v", "5", tmp); err != nil {
		return nil, err
	}
	return os.ReadFile(tmp)
}

// muxNarrationIntoVideo sobrepõe os clipes de narração à faixa de áudio original.
// Cada clipe é posicionado no seu offset. Se extendMs > 0, o vídeo é estendido
// congelando o último frame (para a narração final não ser cortada); nesse caso
// o vídeo é reencodado. Caso contrário, mantém o vídeo intacto (-c:v copy).
func muxNarrationIntoVideo(videoPath string, clips []narrationClip, extendMs int, outPath string) error {
	if len(clips) == 0 {
		return fmt.Errorf("nenhum clipe de narração para integrar")
	}

	hasAudio, err := videoHasAudio(videoPath)
	if err != nil {
		return err
	}

	args := []string{"-y", "-i", videoPath}
	for _, c := range clips {
		args = append(args, "-i", c.wavPath)
	}

	var filter strings.Builder
	var mixLabels []string
	if hasAudio {
		// Reduz o volume do áudio original apenas durante as janelas de narração
		// (ducking temporizado), mantendo o diálogo em volume cheio nas demais partes.
		var windows []string
		for _, c := range clips {
			start := float64(c.offsetMs) / 1000.0
			end := start + float64(c.durationMs)/1000.0
			windows = append(windows, fmt.Sprintf("between(t,%.3f,%.3f)", start, end))
		}
		filter.WriteString(fmt.Sprintf("[0:a]volume=%.2f:enable='%s'[a0];",
			duckVolume, strings.Join(windows, "+")))
		mixLabels = append(mixLabels, "[a0]")
	}
	for i, c := range clips {
		input := i + 1 // índice do input (0 é o vídeo)
		label := fmt.Sprintf("d%d", i)
		filter.WriteString(fmt.Sprintf("[%d:a]adelay=%d|%d[%s];", input, c.offsetMs, c.offsetMs, label))
		mixLabels = append(mixLabels, "["+label+"]")
	}
	filter.WriteString(fmt.Sprintf("%samix=inputs=%d:normalize=0:dropout_transition=0[aout]",
		strings.Join(mixLabels, ""), len(mixLabels)))

	filterComplex := filter.String()

	if extendMs > 0 {
		// Estende o vídeo congelando o último frame até a narração final terminar.
		// Requer reencode do vídeo (não é mais -c:v copy).
		extendSec := float64(extendMs) / 1000.0
		filterComplex += fmt.Sprintf(";[0:v]tpad=stop_mode=clone:stop_duration=%.3f[v]", extendSec)

		args = append(args,
			"-filter_complex", filterComplex,
			"-map", "[v]", "-map", "[aout]",
			"-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p",
			"-c:a", "aac", "-movflags", "+faststart", outPath,
		)
		return runFFmpeg(args...)
	}

	// Vídeo cabe na duração original: mantém o vídeo intacto (-c:v copy).
	args = append(args,
		"-filter_complex", filterComplex,
		"-map", "0:v", "-map", "[aout]",
		"-c:v", "copy", "-c:a", "aac",
	)
	if hasAudio {
		// Com áudio original (que cobre todo o vídeo), garante que a faixa de áudio
		// mixada não ultrapasse a duração do vídeo. Sem áudio original, -shortest
		// cortaria o vídeo no fim da última narração, então não é aplicado.
		args = append(args, "-shortest")
	}
	args = append(args, "-movflags", "+faststart", outPath)

	return runFFmpeg(args...)
}
