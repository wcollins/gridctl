package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	id := json.RawMessage(`"req-1"`)
	resp := NewErrorResponse(&id, MethodNotFound, "method not found")

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	if resp.ID == nil || string(*resp.ID) != `"req-1"` {
		t.Errorf("ID = %v, want %q", resp.ID, `"req-1"`)
	}
	if resp.Result != nil {
		t.Errorf("Result = %v, want nil", resp.Result)
	}
	if resp.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
	if resp.Error.Code != MethodNotFound {
		t.Errorf("Error.Code = %d, want %d", resp.Error.Code, MethodNotFound)
	}
	if resp.Error.Message != "method not found" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "method not found")
	}
}

func TestNewErrorResponse_NilID(t *testing.T) {
	resp := NewErrorResponse(nil, ParseError, "parse error")

	if resp.ID != nil {
		t.Errorf("ID = %v, want nil", resp.ID)
	}
	if resp.Error == nil || resp.Error.Code != ParseError {
		t.Errorf("Error.Code = %d, want %d", resp.Error.Code, ParseError)
	}
}

func TestNewSuccessResponse(t *testing.T) {
	id := json.RawMessage(`1`)
	result := map[string]string{"key": "value"}
	resp := NewSuccessResponse(&id, result)

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	if resp.ID == nil || string(*resp.ID) != "1" {
		t.Errorf("ID = %v, want %q", resp.ID, "1")
	}
	if resp.Error != nil {
		t.Errorf("Error = %v, want nil", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("Result = nil, want non-nil")
	}

	var decoded map[string]string
	if err := json.Unmarshal(resp.Result, &decoded); err != nil {
		t.Fatalf("Unmarshal Result: %v", err)
	}
	if decoded["key"] != "value" {
		t.Errorf("Result[key] = %q, want %q", decoded["key"], "value")
	}
}

func TestNewSuccessResponse_NilResult(t *testing.T) {
	id := json.RawMessage(`"2"`)
	resp := NewSuccessResponse(&id, nil)

	if resp.Result != nil {
		t.Errorf("Result = %v, want nil", resp.Result)
	}
}

func TestRequest_JSON_RoundTrip(t *testing.T) {
	id := json.RawMessage(`"req-1"`)
	req := Request{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "tools/list",
		Params:  json.RawMessage(`{"cursor":null}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, "2.0")
	}
	if decoded.Method != "tools/list" {
		t.Errorf("Method = %q, want %q", decoded.Method, "tools/list")
	}
}

func TestResponse_JSON_RoundTrip(t *testing.T) {
	resp := NewSuccessResponse(nil, []string{"a", "b"})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, "2.0")
	}
	if decoded.Error != nil {
		t.Errorf("Error = %v, want nil", decoded.Error)
	}
	if decoded.Result == nil {
		t.Fatal("Result = nil, want non-nil")
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{"ParseError", ParseError, -32700},
		{"InvalidRequest", InvalidRequest, -32600},
		{"MethodNotFound", MethodNotFound, -32601},
		{"InvalidParams", InvalidParams, -32602},
		{"InternalError", InternalError, -32603},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}
