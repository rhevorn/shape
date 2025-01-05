package main

import (
	"encoding/json"
	"net/http"

	"github.com/rhevorn/goshape"
	goshapehttp "github.com/rhevorn/goshape/http"
)

type CreateUserRequest struct {
	Name  string
	Email string
}

var createUserSchema = goshape.Object[CreateUserRequest](
	goshape.Field("name", goshape.String().Trim().Min(2), func(request *CreateUserRequest, value string) {
		request.Name = value
	}),
	goshape.Field("email", goshape.String().Trim().Email(), func(request *CreateUserRequest, value string) {
		request.Email = value
	}),
).Strict()

func createUser(response http.ResponseWriter, request *http.Request) {
	input, err := goshapehttp.DecodeJSON(request, createUserSchema)
	if err != nil {
		if goshapehttp.WriteValidationError(response, http.StatusUnprocessableEntity, err) {
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
