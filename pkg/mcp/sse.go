package mcp

import (
	"fmt"
	"net/http"
)

// SSEServer handles legacy SSE connections at /sse for backward-compatibility negotiation.
// Clients using the deprecated HTTP+SSE transport receive a negotiation response
// directing them to use the Streamable HTTP transport at POST /mcp.
type SSEServer struct {
	gateway *Gateway
}

// NewSSEServer creates a new SSE server.
func NewSSEServer(gateway *Gateway) *SSEServer {
	return &SSEServer{gateway: gateway}
}

// ServeHTTP handles GET /sse — sends a negotiation event directing the client
// to use the Streamable HTTP transport and closes the connection.
func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send a single negotiation event directing the client to the Streamable HTTP endpoint.
	// Clients should POST an initialize request to /mcp to start a new session.
	fmt.Fprint(w, "event: endpoint\n")
	fmt.Fprint(w, "data: POST /mcp\n\n")
	flusher.Flush()
}

// HandleMessage returns 410 Gone for the legacy /message endpoint.
// Clients should use the Streamable HTTP transport at /mcp instead.
func (s *SSEServer) HandleMessage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "legacy SSE transport is deprecated; use POST /mcp with Streamable HTTP", http.StatusGone)
}

// Close is a no-op retained for interface compatibility.
func (s *SSEServer) Close() {}

// SessionCount returns 0; session management has moved to StreamableHTTPServer.
func (s *SSEServer) SessionCount() int { return 0 }
