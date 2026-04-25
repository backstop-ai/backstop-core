package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/slack-go/slack"
)

// handleSlackEventNoopVerifier creates a verifier but never calls Ensure(),
// so the signature is never actually checked. This satisfies a naive "is
// NewSecretsVerifier present?" check but provides no security.
func handleSlackEventNoopVerifier(w http.ResponseWriter, r *http.Request, signingSecret string) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Create verifier but never call Ensure() — noop
	_, _ = slack.NewSecretsVerifier(r.Header, signingSecret)

	// ruleid: slotly-002
	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
