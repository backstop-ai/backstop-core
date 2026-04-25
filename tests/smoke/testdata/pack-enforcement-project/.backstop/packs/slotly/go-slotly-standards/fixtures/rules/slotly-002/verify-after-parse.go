package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/slack-go/slack"
)

// handleSlackEventWrongOrder verifies AFTER parsing — the payload is already
// trusted by the time verification runs.
func handleSlackEventWrongOrder(w http.ResponseWriter, r *http.Request, signingSecret string) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// ruleid: slotly-002
	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	// Verification happens too late — payload already parsed and potentially acted on
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

	w.WriteHeader(http.StatusOK)
}
