package openapi_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/openapi"
)

func TestJSONRequestBody(t *testing.T) {
	body, err := openapi.JSONRequestBody(shape.String().Min(1), true)
	if err != nil {
		t.Fatal(err)
	}
	content := body["content"].(map[string]any)
	schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
	if _, exists := schema["$schema"]; exists {
		t.Fatal("OpenAPI schema contains root JSON Schema dialect")
	}
	if schema["minLength"] != 1 {
		t.Fatalf("schema = %#v", schema)
	}
	got, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile(filepath.Join("testdata", "request-body.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("request body differs from golden\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestJSONResponse(t *testing.T) {
	response, err := openapi.JSONResponse("ok", shape.Bool())
	if err != nil {
		t.Fatal(err)
	}
	if response["description"] != "ok" {
		t.Fatalf("response = %#v", response)
	}
}
