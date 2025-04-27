package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
	"github.com/rhevorn/shape/openapi"
)

type Request struct {
	Name  string   `json:"name" shape:"notempty,maxlength=50"`
	Age   int      `json:"age" shape:"min=18,max=120"`
	Email *string  `json:"email" shape:"email"`
	Tags  []string `json:"tags" shape:"notempty,unique"`
}

type Transformed struct {
	Name string `json:"name" shape:"trim"`
}

func main() {
	schema := shape.Struct[Request]()

	lowLevel, err := shape.ExportDocument(schema)
	printJSON("shape.ExportDocument", lowLevel, err)

	document, err := jsonschema.Export(schema)
	if err != nil {
		panic(err)
	}
	bytes, err := document.Bytes()
	fmt.Printf("jsonschema.Document.Bytes: %s error=%v\n", bytes, err)

	openAPISchema, err := openapi.Schema(schema)
	printJSON("openapi.Schema", openAPISchema, err)

	requestBody, err := openapi.JSONRequestBody(schema, true)
	printJSON("openapi.JSONRequestBody", requestBody, err)

	response, err := openapi.JSONResponse("validated request", schema)
	printJSON("openapi.JSONResponse", response, err)

	_, err = jsonschema.Export(shape.Struct[Transformed]())
	var unsupported *shape.UnsupportedSchemaError
	fmt.Printf("unsupported transform: typed=%t feature=%q\n",
		errors.As(err, &unsupported), unsupportedFeature(unsupported))
}

func printJSON(name string, value any, err error) {
	encoded, marshalErr := json.Marshal(value)
	if err == nil {
		err = marshalErr
	}
	fmt.Printf("%s: %s error=%v\n", name, encoded, err)
}

func unsupportedFeature(err *shape.UnsupportedSchemaError) string {
	if err == nil {
		return ""
	}
	return err.Feature
}
