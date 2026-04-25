package handlers

import (
	"encoding/json"
	"io"
	"net/http"
)

// handleSlackEventBad parses the payload without any signature verification.
func handleSlackEventBad(w http.ResponseWriter, r *http.Request) {
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

	w.WriteHeader(http.StatusOK)
}
