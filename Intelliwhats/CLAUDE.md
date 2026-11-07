# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

IntelliWhats is a Go WhatsApp bot that adds AI to incoming media: it describes images and stickers, transcribes voice messages, and produces a synchronized **audiodescription of videos** (returns the original video with a spoken narration mixed into its audio track). It connects as a WhatsApp companion device (via the [whatsmeow](https://go.mau.fi/whatsmeow) library), listens for incoming messages, and replies in the same chat.

The Go module lives in `Intelliwhats/` (this directory). The **git repository root is one level up** (`../`), which holds `README.md`, `docker-compose.yml`, `.env` / `.env.example`, and the `data/` volume. Commit from the parent directory.

## Commands

```bash
# Build (pure Go, no C compiler needed)
CGO_ENABLED=0 go build -o intelliwhats .

# Run locally (requires env vars set, see below)
go build -o intelliwhats . && ./intelliwhats

go vet ./...
go fmt ./...

# Run via Docker Compose (from the repo root, ../)
docker-compose up -d       # start
docker-compose logs -f     # follow logs
docker-compose down        # stop
```

There is no test suite in this repo.

**`ffmpeg` and `ffprobe` are required** for the video audiodescription feature (extracting duration/dimensions, converting the TTS PCM to WAV, mixing narration, and re-encoding when extending the video). The Docker runtime installs them via `apk add ffmpeg` (the Alpine `ffmpeg` package includes both binaries). For local dev they must be on `PATH`. `main.go` logs a warning at startup if they are missing (the other features still work without them).

### Required environment variables
- `GROK_API_KEY`, `GEMINI_API_KEY` — required always; `config.go`'s `init()` calls `log.Fatal` if either is missing.
- `PHONE_NUMBER` — required only on first pairing (e.g. `5511999999999`).
- `DB_PATH` — optional, defaults to `./login/whatsmeow.db` (Docker sets `/data/whatsmeow.db`).

First login prints a **pairing code** (not a QR) to the log; enter it under WhatsApp → Linked Devices → Link with phone number. Session state persists in the sqlite DB, so subsequent starts reconnect automatically.

## Architecture

Everything except `models/` is in a single flat `package main`. State is shared through package-level globals declared in `config.go`: `client` (whatsmeow), `db` (`*sql.DB`), `dbContainer`, and a reused `httpClient`.

**Startup (`main.go`)** creates the temp dir → `setupDatabase` → `initializeClient` → `authenticateClient` → blocks on SIGINT/SIGTERM. Auth and DB/client wiring live in `auth.go`.

**Message flow:**
1. `eventHandler` (`handlers.go`) receives all whatsmeow events; `*events.Message` from others (skips `IsFromMe`) is dispatched to `handleMessage` in a goroutine.
2. `handleMessage` branches by media type → `handleImageMessage`, `handleStickerMessage`, `handleAudioMessage`, `handleVideoMessage` (`video.go`), or `handleTextMessage`.
3. Media is downloaded via `client.Download`, sent to an AI provider, and the result is returned — text replies via `replyToMessage`, and the audiodescribed **video via `sendVideoReply`** (`handlers.go`), which uploads with `client.Upload(MediaVideo)` and builds a `waE2E.VideoMessage`. This is the only path that sends media back rather than text.

**AI providers — two backends, selectable per chat:**
- **Grok** (`grok.go`, x.ai) — images/stickers only, sends base64 inline. Default provider.
- **Gemini** (`gemini_image.go`, `gemini_audio.go`, Google) — images/stickers via inline base64; **audio transcription always uses Gemini** regardless of the chat preference (Grok has no audio path).
- Images/stickers respect the per-chat preference; audio does not.

**Per-chat preference (`preferences.go`):** stored in the `user_preferences` SQLite table keyed by `chat_jid`. `APIProvider` is an int enum (`APIGrok=1`, `APIGemini=2`); `getAPIPreference` falls back to `APIGrok` when no row exists or on error. Users switch with the `!api` command.

**Gemini audio** uses a resumable file-upload flow (init upload → upload bytes → request transcription referencing the file URI), unlike images which are inlined. See `gemini_audio.go`. The upload chain (`initializeGeminiUpload`, `uploadFileToGemini` — which returns the full `models.GeminiFile`) is mimetype-agnostic and **reused by the video path**.

**Video audiodescription (`video.go`, `gemini_video.go`, `gemini_tts.go`, `ffmpeg.go`)** — always uses Gemini regardless of the chat's `!api` preference (Grok can't process video). The pipeline in `handleVideoMessage` → `buildAudioDescribedVideo`:
1. Upload the video (reusing the resumable chain) and **poll `waitForGeminiFileActive`** until the Gemini File API reports `ACTIVE` (video needs processing before use); the file is deleted afterwards (`deleteGeminiFile`, best-effort).
2. `requestGeminiADSegments` asks Gemini for **structured JSON** (`responseMimeType: application/json` + schema) — an array of `{start_ms, text}` segments placed in the dialogue's pauses. `sanitizeSegments` drops invalid ones, sorts by time, and caps at `maxADSegments` (40). If the JSON path fails, it **falls back** to a single narration (`requestGeminiVideoDescription`).
3. Each segment is synthesized to speech via **`synthesizeSpeech` (`gemini_tts.go`, model `GEMINI_TTS_MODEL`)**, which returns raw PCM (s16le, 24kHz, mono). TTS calls are **globally throttled** (`ttsThrottle`, ~7s apart) and **retry on HTTP 429** honoring the API's `retryDelay` (the free tier allows only ~10 TTS req/min). The loop skips segments that overlap the previous narration or start past the video's end (before spending a TTS call).
4. `ffmpeg.go` converts PCM→WAV (`pcmToWavFile`) and `muxNarrationIntoVideo` builds one `filter_complex`: **time-gated ducking** of the original audio during each narration window (`volume=…:enable='between(t,…)'`) + `adelay` per clip + `amix`. Normally keeps the video with `-c:v copy` (+`-shortest` when there's original audio). If the last narration runs past the video, it passes `extendMs > 0` to **freeze the last frame** (`tpad=stop_mode=clone`) so the narration isn't cut — this branch re-encodes with `libx264`.

**Request/response structs** for both APIs live in `models/` (`grok_models.go`, `gemini_models.go`), imported as `intelliwhats/models`.

**Prompts are centralized** in `config.go`: `GetImageDescriptionPrompt(language)`, `GetAudioTranscriptionPrompt(language)`, and `GetVideoADSegmentsPrompt(language)` (the video prompt insists on very short per-segment text so the spoken narration fits inside dialogue pauses). Language defaults to `pt-BR` (currently hardcoded at the call sites). Change prompts here, not in the provider files.

## Conventions

- Code comments, log messages, and user-facing bot replies are in **Portuguese (pt-BR)**. Match this when editing.
- Chat commands handled in `handleTextMessage`: `!ping`, `!api` (show current), `!api 1` (Grok), `!api 2` (Gemini).
- Model IDs and API URLs are constants at the top of `config.go` (`GROK_MODEL`, `GEMINI_MODEL`, `GROK_API_URL`, `GEMINI_TTS_MODEL`, `GEMINI_TTS_VOICE`, etc.).
- Video processing is Gemini-only and shells out to `ffmpeg`/`ffprobe` via `os/exec` (wrappers in `ffmpeg.go`); temp files go under `tempDir` (`./temp`) and are cleaned with `defer os.Remove`.
