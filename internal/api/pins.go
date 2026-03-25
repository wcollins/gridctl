package api

import (
	"net/http"
	"strings"
)

// handlePins routes all /api/pins requests.
func (s *Server) handlePins(w http.ResponseWriter, r *http.Request) {
	if s.pinStore == nil {
		writeJSONError(w, "Pin store not available", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/pins")
	path = strings.TrimPrefix(path, "/")

	// Split into segments: "", "{server}", "{server}/approve"
	segments := strings.SplitN(path, "/", 3)
	first := segments[0]
	second := ""
	if len(segments) > 1 {
		second = segments[1]
	}

	switch {
	case first == "" && r.Method == http.MethodGet:
		s.handleListPins(w, r)
	case first != "" && second == "" && r.Method == http.MethodGet:
		s.handleGetServerPins(w, r, first)
	case first != "" && second == "approve" && r.Method == http.MethodPost:
		s.handleApprovePins(w, r, first)
	case first != "" && second == "" && r.Method == http.MethodDelete:
		s.handleResetPins(w, r, first)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListPins returns all servers' pin records.
// GET /api/pins
func (s *Server) handleListPins(w http.ResponseWriter, r *http.Request) {
	servers := s.pinStore.GetAll()
	writeJSON(w, servers)
}

// handleGetServerPins returns the pin record for a single server.
// GET /api/pins/{server}
func (s *Server) handleGetServerPins(w http.ResponseWriter, r *http.Request, serverName string) {
	sp, ok := s.pinStore.GetServer(serverName)
	if !ok {
		writeJSONError(w, "No pins found for server: "+serverName, http.StatusNotFound)
		return
	}
	writeJSON(w, sp)
}

// handleApprovePins re-pins the current tool definitions for a server, clearing drift.
// POST /api/pins/{server}/approve
func (s *Server) handleApprovePins(w http.ResponseWriter, r *http.Request, serverName string) {
	// Verify the server has existing pins before approving.
	if _, ok := s.pinStore.GetServer(serverName); !ok {
		writeJSONError(w, "No pins found for server: "+serverName, http.StatusNotFound)
		return
	}

	// Fetch the current live tools from the gateway router.
	client := s.gateway.Router().GetClient(serverName)
	if client == nil {
		writeJSONError(w, "Server not found in gateway: "+serverName, http.StatusNotFound)
		return
	}
	tools := client.Tools()

	if err := s.pinStore.Approve(serverName, tools); err != nil {
		writeJSONError(w, "Failed to approve pins: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"server":     serverName,
		"tool_count": len(tools),
		"status":     "approved",
	})
}

// handleResetPins deletes the pin record for a server.
// DELETE /api/pins/{server}
func (s *Server) handleResetPins(w http.ResponseWriter, r *http.Request, serverName string) {
	if _, ok := s.pinStore.GetServer(serverName); !ok {
		writeJSONError(w, "No pins found for server: "+serverName, http.StatusNotFound)
		return
	}

	if err := s.pinStore.Reset(serverName); err != nil {
		writeJSONError(w, "Failed to reset pins: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
