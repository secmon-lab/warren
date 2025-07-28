package websocket

import "net/http"

// CheckOriginExported exports checkOrigin for testing
func (h *Handler) CheckOriginExported(r *http.Request) bool {
	return h.checkOrigin(r)
}
