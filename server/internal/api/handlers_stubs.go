package api

import (
	"net/http"
)

// handleHealth returns the server health status
func (server *Server) handleHealth(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}
