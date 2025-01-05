package openapi_test

import (
	"testing"

	"github.com/rhevorn/goshape"
	"github.com/rhevorn/goshape/openapi"
)

func TestJSONRequestBody(t *testing.T) {
	body, err := openapi.JSONRequestBody(goshape.String().Min(1), true)
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
}

func TestJSONResponse(t *testing.T) {
	response, err := openapi.JSONResponse("ok", goshape.Bool())
	if err != nil {
		t.Fatal(err)
	}
	if response["description"] != "ok" {
		t.Fatalf("response = %#v", response)
	}
}
