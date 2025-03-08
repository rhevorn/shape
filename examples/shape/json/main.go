package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhevorn/shape"
)

type Request struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
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

	target := Request{Name: "unchanged", Age: 99}
	err = requestSchema.BindJSON(&target, []byte(`{"name":"new","age":10}`), strictJSON)
	fmt.Printf("failed bind keeps target: %#v error=%v\n", target, err)

	_, err = requestSchema.ParseJSONReader(strings.NewReader(`{"name":"Pong","age":20}`), shape.JSONOptions{MaxBytes: 4})
	fmt.Println("too large:", errors.Is(err, shape.ErrJSONTooLarge))
}
