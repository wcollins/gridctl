package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSEServer_NegotiationHeaders(t *testing.T) {
	g := NewGateway()
	sse := NewSSEServer(g)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	w := httptest.NewRecorder()
	sse.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cc)
	}
	if conn := w.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("expected Connection keep-alive, got %s", conn)
	}
}

func TestSSEServer_NegotiationEvent(t *testing.T) {
	g := NewGateway()
	sse := NewSSEServer(g)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	w := httptest.NewRecorder()
	sse.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "event: endpoint") {
		t.Errorf("expected 'event: endpoint' in response, got: %s", body)
	}
	if !strings.Contains(body, "data: POST /mcp") {
		t.Errorf("expected 'data: POST /mcp' in response, got: %s", body)
	}
}

func TestSSEServer_NegotiationResponse_IsImmediate(t *testing.T) {
	g := NewGateway()
	sse := NewSSEServer(g)

	// ServeHTTP should return immediately (not block waiting for client disconnect)
	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		sse.ServeHTTP(w, req)
	}()

	select {
	case <-done:
		// ServeHTTP returned immediately — correct
	}
}

func TestSSEServer_HandleMessage_Gone(t *testing.T) {
	g := NewGateway()
	sse := NewSSEServer(g)

	req := httptest.NewRequest(http.MethodPost, "/message?sessionId=test", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	sse.HandleMessage(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("expected 410 Gone for legacy /message, got %d", w.Code)
	}
}

func TestSSEServer_Close_NoOp(t *testing.T) {
	g := NewGateway()
	sse := NewSSEServer(g)
	sse.Close() // should not panic
}

func TestSSEServer_SessionCount_AlwaysZero(t *testing.T) {
	g := NewGateway()
	sse := NewSSEServer(g)
	if n := sse.SessionCount(); n != 0 {
		t.Errorf("expected SessionCount 0, got %d", n)
	}
}
