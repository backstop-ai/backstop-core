package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/slack-go/slack"
)

// handleSlackEvent shows the correct pattern: verify signature before parsing.
func handleSlackEvent(w http.ResponseWriter, r *http.Request, signingSecret string) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := sv.Write(bodyBytes); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := sv.Ensure(); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
