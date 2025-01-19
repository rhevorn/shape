package goshapehttp_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhevorn/goshape"
	goshapehttp "github.com/rhevorn/goshape/http"
)

func TestDecodeJSON(t *testing.T) {
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Pong"}`))
	type payload struct{ Name string }
	schema := goshape.Object[payload](
		goshape.Field("name", goshape.String().Min(1), func(value *payload, name string) { value.Name = name }),
	).Strict()
	got, err := goshapehttp.DecodeJSON(request, schema)
	if err != nil || got.Name != "Pong" {
		t.Fatalf("DecodeJSON = %#v, %v", got, err)
	}
}

func TestBodyLimitAndValidationResponse(t *testing.T) {
	request := httptest.NewRequest("POST", "/", strings.NewReader(`"too long"`))
	_, err := goshapehttp.DecodeJSONLimit(request, goshape.String(), 2)
	if !errors.Is(err, goshapehttp.ErrBodyTooLarge) {
		t.Fatalf("error = %v", err)
	}

	_, validationErr := goshape.String().Min(2).Parse("x")
	response := httptest.NewRecorder()
	if !goshapehttp.WriteValidationError(response, 422, validationErr) {
		t.Fatal("validation error was not handled")
	}
	if response.Code != 422 || !strings.Contains(response.Body.String(), `"too_small"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if goshapehttp.WriteValidationError(httptest.NewRecorder(), 400, errors.New("plain")) {
		t.Fatal("plain error was handled as a validation error")
	}
	if _, err := goshapehttp.DecodeJSON[string](nil, goshape.String()); err == nil {
		t.Fatal("nil request was accepted")
	}
	if _, err := goshapehttp.DecodeJSON[string](&http.Request{}, goshape.String()); err == nil {
		t.Fatal("nil request body was accepted")
	}
	request = &http.Request{Body: io.NopCloser(strings.NewReader(`"ok"`))}
	if got, err := goshapehttp.DecodeJSONLimit(request, goshape.String(), int64(1<<63-1)); err != nil || got != "ok" {
		t.Fatalf("maximum limit = %q, %v", got, err)
	}
}
