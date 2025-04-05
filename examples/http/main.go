package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/rhevorn/shape"
)

type Request struct {
	Name string `json:"name" shape:"trim,notempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	var request Request
	err := shape.BindJSONReaderContext(r.Context(), &request, r.Body, shape.JSONOptions{
		DisallowUnknownFields: true,
		MaxBytes:              1 << 20,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, shape.ErrJSONTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(request)
}

func main() {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":" Pong "}`))
	response := httptest.NewRecorder()
	handler(response, request)
	fmt.Print(response.Body.String())
}
