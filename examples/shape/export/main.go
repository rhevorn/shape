package main

import (
	"encoding/json"
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
	"github.com/rhevorn/shape/openapi"
)

type Request struct {
	Name string `json:"name" shape:"notempty,maxlength=50"`
	Age  int    `json:"age" shape:"min=18"`
}

func main() {
	schema := shape.Struct[Request]()
	document, err := jsonschema.Export(schema)
	if err != nil {
		panic(err)
	}
	body, _ := json.MarshalIndent(document, "", "  ")
	fmt.Println(string(body))

	requestBody, err := openapi.JSONRequestBody(schema, true)
	if err != nil {
		panic(err)
	}
	body, _ = json.MarshalIndent(requestBody, "", "  ")
	fmt.Println(string(body))
}
