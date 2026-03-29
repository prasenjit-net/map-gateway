package admin

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/prasenjit-net/mcp-gateway/config"
)

const openAICompletionsURL = "https://api.openai.com/v1/chat/completions"

type chatHandler struct {
	config *config.Config
}

// chatConfig is the safe subset of config exposed to the browser.
type chatConfig struct {
	Model  string `json:"model"`
	HasKey bool   `json:"hasKey"`
}

// GetConfig returns the server-configured model and whether an API key is set.
// The API key itself is never returned.
func (h *chatHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatConfig{
		Model:  h.config.OpenAIModel,
		HasKey: h.config.OpenAIAPIKey != "",
	})
}

// Completions proxies a chat/completions request to OpenAI, injecting the
// server-side API key so it is never exposed to the browser.
// The request body and response (including SSE streams) are piped verbatim.
func (h *chatHandler) Completions(w http.ResponseWriter, r *http.Request) {
	if h.config.OpenAIAPIKey == "" {
		jsonError(w, "OpenAI API key not configured on server", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, openAICompletionsURL, r.Body)
	if err != nil {
		jsonError(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.config.OpenAIAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, "upstream request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy upstream headers (Content-Type, transfer-encoding, etc.) and status.
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Stream the body back; flush after each write so SSE chunks arrive promptly.
	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n]) //nolint:errcheck
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return
		}
	}
}
