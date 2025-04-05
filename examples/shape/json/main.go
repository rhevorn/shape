package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhevorn/shape"
)

type Request struct {
	Name string `json:"name" shape:"trim,notempty"`
	Age  int    `json:"age" shape:"min=18"`
}

var requestSchema = shape.New[Request](
	shape.String("Name").Trim().NotEmpty(),
	shape.Int("Age").Min(18),
)

var strictJSON = shape.JSONOptions{
	DisallowUnknownFields: true,
	MaxBytes:              1 << 20,
}

func main() {
	request, err := requestSchema.ParseJSONReader(
		strings.NewReader(`{"name":" Pong ","age":20}`),
		strictJSON,
	)
	fmt.Printf("parsed: %#v error=%v\n", request, err)

	_, err = requestSchema.ParseJSON([]byte(`{"name":"new","age":20,"extra":true}`), strictJSON)
	fmt.Printf("unknown field rejected: %v\n", err)

	target := Request{Name: "keep", Age: 99}
	err = shape.BindJSON(&target, []byte(`{"name":" changed ","age":10}`), strictJSON)
	fmt.Printf("atomic bind: value=%#v error=%v\n", target, err)

	_, err = requestSchema.ParseJSONReader(strings.NewReader(`{"name":"Pong","age":20}`), shape.JSONOptions{MaxBytes: 4})
	fmt.Println("too large:", errors.Is(err, shape.ErrJSONTooLarge))
}
