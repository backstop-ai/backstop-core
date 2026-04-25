package slackhandler

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/slack/events", h.HandleEvent)
}
