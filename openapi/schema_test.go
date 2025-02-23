package openapi_test

import (
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/openapi"
)

func TestJSONRequestBody(t *testing.T) {
	type Request struct {
		Name string `json:"name" shape:"minlength=1"`
	}
	body, err := openapi.JSONRequestBody(shape.Struct[Request](), true)
	if err != nil {
		t.Fatal(err)
	}
	if body["required"] != true {
		t.Fatalf("body = %#v", body)
	}
}

func TestJSONResponse(t *testing.T) {
	type Response struct {
		OK bool `json:"ok"`
	}
	response, err := openapi.JSONResponse("ok", shape.Struct[Response]())
	if err != nil || response["description"] != "ok" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}
