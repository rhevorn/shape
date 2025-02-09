package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rhevorn/shape"
)

// Struct tags are optional. Prefer Fields/Object for most schemas; use
// MustStruct / Bind when a simple DTO is enough.
type CreateUserRequest struct {
	Name    string        `json:"name" shape:"trim,min=2,max=50,label='姓名'"`
	Email   string        `json:"email" shape:"trim,email,label='邮箱'"`
	Age     int           `json:"age" shape:"min=18,max=120,label='年龄'"`
	Timeout time.Duration `json:"timeout" shape:"coerce,optional"`
}

var createUser = shape.MustStruct[CreateUserRequest]().Strict()

func main() {
	user, err := shape.Parse(createUser, `{
		"name": " Pong ",
		"email": "pong@example.com",
		"age": 30,
		"timeout": "3s"
	}`)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", user)

	// Bind uses the same type-cached MustStruct schema (Strict copy per call).
	ctx := context.Background()
	body := strings.NewReader(`{"name":"Ada","email":"ada@example.com","age":36}`)
	const maxBody = 1 << 20 // 1 MiB

	var req CreateUserRequest
	err = shape.BindReaderLimitContext(ctx, &req, body, maxBody)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", req)
}
