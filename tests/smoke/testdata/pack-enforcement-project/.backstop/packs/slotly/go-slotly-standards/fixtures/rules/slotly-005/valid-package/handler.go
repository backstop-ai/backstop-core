package slackhandler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/slack-go/slack"
)

type Handler struct {
	signingSecret string
}

func (h *Handler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sv, err := slack.NewSecretsVerifier(r.Header, h.signingSecret)
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
	_ = json.Unmarshal(bodyBytes, &payload)
	w.WriteHeader(http.StatusOK)
}
