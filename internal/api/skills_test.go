package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gridctl/gridctl/pkg/mcp"
	"github.com/gridctl/gridctl/pkg/registry"
)

// --- Skills Sources Endpoints ---

func TestHandleSkills_SourcesList_Empty(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/sources", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result []SkillSourceStatus
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty sources, got %d", len(result))
	}
}

func TestHandleSkills_SourceAdd_MissingRepo(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/skills/sources", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if !strings.Contains(errResp["error"], "repo") {
		t.Errorf("expected error about repo, got %q", errResp["error"])
	}
}

func TestHandleSkills_SourceAdd_InvalidJSON(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	body := strings.NewReader(`{invalid}`)
	req := httptest.NewRequest(http.MethodPost, "/api/skills/sources", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSkills_UpdatesEmpty(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/updates", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var summary UpdateSummary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if summary.Available != 0 {
		t.Errorf("expected 0 updates, got %d", summary.Available)
	}
}

func TestHandleSkills_SourceRemove_NotFound(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodDelete, "/api/skills/sources/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSkills_SourceCheck_NotFound(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/sources/nonexistent/check", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSkills_SourceUpdate_NotFound(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/sources/nonexistent/update", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSkills_SourcePreview_MissingRepo(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/sources/newrepo/preview", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSkills_NoRegistry(t *testing.T) {
	gateway := mcp.NewGateway()
	srv := NewServer(gateway, nil)
	// No registry set

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/sources", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSkills_RoutingMethodNotAllowed(t *testing.T) {
	srv, _ := setupRegistryTestServer(t)

	handler := srv.Handler()

	// GET on sources should work
	req := httptest.NewRequest(http.MethodGet, "/api/skills/sources", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET sources, got %d", rec.Code)
	}
}

func TestStoreDir(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(dir)

	if store.Dir() != dir {
		t.Errorf("expected Dir() = %q, got %q", dir, store.Dir())
	}
}
