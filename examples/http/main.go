package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type CreateUserRequest struct {
	Name  string   `json:"name" shape:"trim,notempty,maxlength=50"`
	Email string   `json:"email" shape:"trim,email"`
	Roles []string `json:"roles" shape:"notempty,max=5,unique"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var request CreateUserRequest
	err := shape.BindJSONReaderContext(r.Context(), &request, r.Body, shape.JSONOptions{
		DisallowUnknownFields: true,
		MaxBytes:              1 << 10,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(request)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	var validationError *validate.Error
	switch {
	case errors.Is(err, shape.ErrJSONTooLarge):
		status = http.StatusRequestEntityTooLarge
	case errors.As(err, &validationError):
		status = http.StatusUnprocessableEntity
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}

func main() {
	run(`{"name":" Pong ","email":"pong@example.com","roles":["admin"]}`)
	run(`{"name":"","email":"bad","roles":[]}`)
	run(`{"name":"Pong","email":"pong@example.com","roles":["admin"],"extra":true}`)
	run(`{"name":"` + strings.Repeat("x", 2048) + `","email":"pong@example.com","roles":["admin"]}`)
}

func run(body string) {
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	response := httptest.NewRecorder()
	createUser(response, request)
	fmt.Printf("status=%d body=%s", response.Code, response.Body.String())
}
