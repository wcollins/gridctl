package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewStructuredLogger_JSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  slog.LevelInfo,
		Format: FormatJSON,
		Output: &buf,
	}

	logger := NewStructuredLogger(cfg)
	logger.Info("test message", "key", "value")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Errorf("expected key 'value', got %v", entry["key"])
	}
}

func TestNewStructuredLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:     slog.LevelInfo,
		Format:    FormatJSON,
		Output:    &buf,
		Component: "test-component",
	}

	logger := NewStructuredLogger(cfg)
	logger.Info("test message")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["component"] != "test-component" {
		t.Errorf("expected component 'test-component', got %v", entry["component"])
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected LogFormat
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"text", FormatText},
		{"TEXT", FormatText},
		{"pretty", FormatText},
		{"unknown", FormatJSON}, // defaults to json
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithTraceID(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  slog.LevelInfo,
		Format: FormatJSON,
		Output: &buf,
	}

	logger := NewStructuredLogger(cfg)
	logger = WithTraceID(logger, "trace-123")
	logger.Info("test message")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["trace_id"] != "trace-123" {
		t.Errorf("expected trace_id 'trace-123', got %v", entry["trace_id"])
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  slog.LevelInfo,
		Format: FormatJSON,
		Output: &buf,
	}

	logger := NewStructuredLogger(cfg)
	logger = WithComponent(logger, "my-component")
	logger.Info("test message")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["component"] != "my-component" {
		t.Errorf("expected component 'my-component', got %v", entry["component"])
	}
}

func TestCaller(t *testing.T) {
	file, line := Caller(0)
	if file == "unknown" || file == "" {
		t.Error("expected file to be identified")
	}
	if line == 0 {
		t.Error("expected line to be non-zero")
	}
	// Should be in this test file
	if file != "structured_test.go" {
		t.Errorf("expected file 'structured_test.go', got %s", file)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != slog.LevelInfo {
		t.Errorf("expected default level INFO, got %v", cfg.Level)
	}
	if cfg.Format != FormatJSON {
		t.Errorf("expected default format JSON, got %v", cfg.Format)
	}
	if cfg.AddSource {
		t.Error("expected AddSource to be false by default")
	}
}
