package main

import (
	"encoding/json"
	"net/http"

	"github.com/rhevorn/shape"
	shapehttp "github.com/rhevorn/shape/http"
)

type CreateUserRequest struct {
	Name  string
	Email string
}

var createUserSchema = shape.Object[CreateUserRequest](
	shape.Field("name", shape.String().Trim().Min(2), func(request *CreateUserRequest, value string) {
		request.Name = value
	}),
	shape.Field("email", shape.String().Trim().Email(), func(request *CreateUserRequest, value string) {
		request.Email = value
	}),
).Strict()

func createUser(response http.ResponseWriter, request *http.Request) {
	input, err := shapehttp.DecodeJSON(request, createUserSchema)
	if err != nil {
		if shapehttp.WriteValidationError(response, http.StatusUnprocessableEntity, err) {
			return
		}
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(response).Encode(input)
}

func main() {
	http.HandleFunc("POST /users", createUser)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
