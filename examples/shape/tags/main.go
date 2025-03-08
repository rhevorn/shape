package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rhevorn/shape"
)

type Address struct {
	City string `json:"city" shape:"trim,notempty"`
}

type Request struct {
	Name       string            `json:"name" shape:"ifzero=guest,trim,notempty,maxlength=50"`
	Nickname   *string           `json:"nickname" shape:"ifnull=anonymous"`
	Addresses  []Address         `json:"addresses" shape:"notempty,max=3"`
	Attributes map[string]string `json:"attributes" shape:"notnull,max=10"`
}

var requestSchema = shape.Struct[Request]()

func main() {
	request, err := requestSchema.ParseJSON([]byte(`{
		"name":" Pong ",
		"addresses":[{"city":" Shanghai "}],
		"attributes":{"role":"admin"}
	}`))
	fmt.Printf("parsed: %#v error=%v\n", request, err)

	var bound Request
	err = shape.BindJSONReaderContext(
		context.Background(),
		&bound,
		strings.NewReader(`{"name":" Shape ","addresses":[{"city":" Hangzhou "}],"attributes":{}}`),
	)
	fmt.Printf("bound: %#v error=%v\n", bound, err)
}
