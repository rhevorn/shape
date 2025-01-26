package shapehttp_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhevorn/shape"
	shapehttp "github.com/rhevorn/shape/http"
)

func TestDecodeJSON(t *testing.T) {
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Pong"}`))
	type payload struct{ Name string }
	schema := shape.Object[payload](
		shape.Field("name", shape.String().Min(1), func(value *payload, name string) { value.Name = name }),
	).Strict()
	got, err := shapehttp.DecodeJSON(request, schema)
	if err != nil || got.Name != "Pong" {
		t.Fatalf("DecodeJSON = %#v, %v", got, err)
	}
}

func TestBodyLimitAndValidationResponse(t *testing.T) {
	request := httptest.NewRequest("POST", "/", strings.NewReader(`"too long"`))
	_, err := shapehttp.DecodeJSONLimit(request, shape.String(), 2)
	if !errors.Is(err, shapehttp.ErrBodyTooLarge) {
		t.Fatalf("error = %v", err)
	}

	_, validationErr := shape.String().Min(2).Parse("x")
	response := httptest.NewRecorder()
	if !shapehttp.WriteValidationError(response, 422, validationErr) {
		t.Fatal("validation error was not handled")
	}
	if response.Code != 422 || !strings.Contains(response.Body.String(), `"too_small"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if shapehttp.WriteValidationError(httptest.NewRecorder(), 400, errors.New("plain")) {
		t.Fatal("plain error was handled as a validation error")
	}
	if _, err := shapehttp.DecodeJSON[string](nil, shape.String()); err == nil {
		t.Fatal("nil request was accepted")
	}
	if _, err := shapehttp.DecodeJSON[string](&http.Request{}, shape.String()); err == nil {
		t.Fatal("nil request body was accepted")
	}
	request = &http.Request{Body: io.NopCloser(strings.NewReader(`"ok"`))}
	if got, err := shapehttp.DecodeJSONLimit(request, shape.String(), int64(1<<63-1)); err != nil || got != "ok" {
		t.Fatalf("maximum limit = %q, %v", got, err)
	}
}

func TestValidationResponseIssueLimit(t *testing.T) {
	issues := make([]shape.Issue, 5)
	for index := range issues {
		issues[index] = shape.Issue{Code: shape.CodeInvalidValue, Message: "invalid"}
	}
	response := httptest.NewRecorder()
	if !shapehttp.WriteValidationErrorLimit(response, 400, &shape.ValidationError{Issues: issues}, 3) {
		t.Fatal("validation error was not handled")
	}
	if got := strings.Count(response.Body.String(), `"code"`); got != 3 {
		t.Fatalf("encoded issue count = %d, want 3: %s", got, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), shape.CodeTooManyIssues) {
		t.Fatalf("response lacks truncation issue: %s", response.Body.String())
	}
}
