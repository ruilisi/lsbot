// Package transcription provides voice-to-text via the OpenAI Whisper API.
// It is used by messaging platform adapters to convert audio files
// (voice messages, voice memos) into text before passing them to the agent.
package transcription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	whisperURL     = "https://api.openai.com/v1/audio/transcriptions"
	defaultModel   = "whisper-1"
	requestTimeout = 60 * time.Second
)

// Transcriber calls the Whisper API to convert audio to text.
type Transcriber struct {
	apiKey  string
	baseURL string // overridable for testing / compatible endpoints
	client  *http.Client
}

// New creates a Transcriber with the given OpenAI API key.
// baseURL may be "" to use the default OpenAI endpoint.
func New(apiKey, baseURL string) *Transcriber {
	endpointURL := whisperURL
	if baseURL != "" {
		endpointURL = strings.TrimRight(baseURL, "/") + "/audio/transcriptions"
	}
	return &Transcriber{
		apiKey:  apiKey,
		baseURL: endpointURL,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

// TranscribeFile sends audioPath to Whisper and returns the transcript text.
// Supported formats: mp3, mp4, mpeg, mpga, m4a, wav, webm, ogg.
func (t *Transcriber) TranscribeFile(audioPath string) (string, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("transcription: open audio: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// model field
	if err := w.WriteField("model", defaultModel); err != nil {
		return "", err
	}

	// audio file field
	part, err := w.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", fmt.Errorf("transcription: read audio: %w", err)
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, t.baseURL, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription: request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("transcription: API error %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("transcription: parse response: %w", err)
	}
	return result.Text, nil
}

// IsAudioFile reports whether the file extension is a supported audio format.
func IsAudioFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm", ".ogg", ".oga":
		return true
	}
	return false
}
