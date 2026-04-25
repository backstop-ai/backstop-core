package slackhandler

import (
	"encoding/json"
	"io"
	"net/http"
)

type Handler struct{}

// HandleEvent processes Slack events without any signature verification.
func (h *Handler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &payload)
	w.WriteHeader(http.StatusOK)
}
