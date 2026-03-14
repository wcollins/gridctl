package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gridctl/gridctl/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestHandleStackValidate_ValidYAML(t *testing.T) {
	s := &Server{}
	body := `
name: test
network:
  name: test-net
mcp-servers:
  - name: s1
    image: alpine
    port: 3000
`
	req := httptest.NewRequest(http.MethodPost, "/api/stack/validate", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleStackValidate(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":true`)
}

func TestHandleStackValidate_InvalidYAML(t *testing.T) {
	s := &Server{}
	body := `:::not yaml`
	req := httptest.NewRequest(http.MethodPost, "/api/stack/validate", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleStackValidate(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":false`)
	assert.Contains(t, w.Body.String(), `"severity":"error"`)
}

func TestHandleStackValidate_InvalidStack(t *testing.T) {
	s := &Server{}
	body := `
mcp-servers:
  - name: s1
`
	req := httptest.NewRequest(http.MethodPost, "/api/stack/validate", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleStackValidate(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":false`)
	assert.Contains(t, w.Body.String(), `"errorCount"`)
}

func TestHandleStackSpec_NoStackFile(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/stack/spec", nil)
	w := httptest.NewRecorder()

	s.handleStackSpec(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleStackPlan_NoStackDeployed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/stack/plan", nil)
	w := httptest.NewRecorder()

	s.handleStackPlan(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleStackHealth_NoStackFile(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/stack/health", nil)
	w := httptest.NewRecorder()

	s.handleStackHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"unknown"`)
}

func TestHandleStackExport_NoStackFile(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/stack/export", nil)
	w := httptest.NewRecorder()

	s.handleStackExport(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleStackSecretsMap_NoStackFile(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/stack/secrets-map", nil)
	w := httptest.NewRecorder()

	s.handleStackSecretsMap(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"secrets"`)
	assert.Contains(t, w.Body.String(), `"nodes"`)
}

func TestHandleStackRecipes(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/stack/recipes", nil)
	w := httptest.NewRecorder()

	s.handleStackRecipes(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"rag-pipeline"`)
	assert.Contains(t, w.Body.String(), `"dev-toolbox"`)
}

func TestSanitizeStackSecrets(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{"already vault ref", "API_KEY", "${vault:MY_KEY}", "${vault:MY_KEY}"},
		{"sensitive password", "DB_PASSWORD", "secret123", "${vault:test_DB_PASSWORD}"},
		{"sensitive token", "AUTH_TOKEN", "tok_abc", "${vault:test_AUTH_TOKEN}"},
		{"non-sensitive", "HOST", "localhost", "localhost"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{tc.key: tc.value}
			sanitizeStackSecrets(&config.Stack{
				MCPServers: []config.MCPServer{{Name: "test", Env: env}},
			})
			assert.Equal(t, tc.expected, env[tc.key])
		})
	}
}

func TestAppendUnique(t *testing.T) {
	result := appendUnique([]string{"a", "b"}, "c")
	assert.Equal(t, []string{"a", "b", "c"}, result)

	result = appendUnique([]string{"a", "b"}, "a")
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestHandleStack_Routing(t *testing.T) {
	s := &Server{}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"validate POST", http.MethodPost, "/api/stack/validate", http.StatusOK},
		{"plan GET no stack", http.MethodGet, "/api/stack/plan", http.StatusServiceUnavailable},
		{"health GET", http.MethodGet, "/api/stack/health", http.StatusOK},
		{"spec GET no stack", http.MethodGet, "/api/stack/spec", http.StatusServiceUnavailable},
		{"export GET no stack", http.MethodGet, "/api/stack/export", http.StatusServiceUnavailable},
		{"secrets-map GET", http.MethodGet, "/api/stack/secrets-map", http.StatusOK},
		{"recipes GET", http.MethodGet, "/api/stack/recipes", http.StatusOK},
		{"unknown path", http.MethodGet, "/api/stack/unknown", http.StatusMethodNotAllowed},
		{"validate wrong method", http.MethodGet, "/api/stack/validate", http.StatusMethodNotAllowed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body *strings.Reader
			if tc.method == http.MethodPost {
				body = strings.NewReader(`name: test`)
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			w := httptest.NewRecorder()

			s.handleStack(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
