package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rhevorn/shape"
)

type CreateUserRequest struct {
	Name  string
	Email string
}

var createUserSchema = func() shape.ObjectSchema[CreateUserRequest] {
	f := shape.Fields[CreateUserRequest]()
	return shape.Object(
		f.Str("name", "姓名").Trim().Min(2).Set(func(request *CreateUserRequest, value string) {
			request.Name = value
		}),
		f.Email("email", "邮箱").Trim().Set(func(request *CreateUserRequest, value string) {
			request.Email = value
		}),
	).Strict()
}()

func createUser(w http.ResponseWriter, r *http.Request) {
	input, err := shape.ParseReaderLimitContext(r.Context(), createUserSchema, r.Body, 1<<20)
	if err != nil {
		var validation *shape.ValidationError
		if errors.As(err, &validation) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": validation.Issues})
			return
		}
		if errors.Is(err, shape.ErrJSONTooLarge) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(input)
}

func main() {
	http.HandleFunc("POST /users", createUser)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
