package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rhevorn/shape"
)

type Request struct {
	Name string `json:"name" shape:"trim,notempty"`
	Age  int    `json:"age" shape:"min=18"`
}

var explicitSchema = shape.New[Request](
	shape.Field("Name", shape.String().Trim().NotEmpty()),
	shape.Field("Age", shape.Int().Min(18)),
)

var taggedSchema = shape.Struct[Request]()

var strict = shape.JSONOptions{
	DisallowUnknownFields: true,
	MaxBytes:              1 << 20,
}

func main() {
	ctx := context.Background()
	data := []byte(`{"name":" Pong ","age":20}`)

	a, err := explicitSchema.ParseJSON(data, strict)
	b, errB := explicitSchema.ParseJSONContext(ctx, data, strict)
	c, errC := explicitSchema.ParseJSONReader(strings.NewReader(string(data)), strict)
	d, errD := explicitSchema.ParseJSONReaderContext(ctx, strings.NewReader(string(data)), strict)
	fmt.Printf("schema parse: %q/%q/%q/%q errors=%v,%v,%v,%v\n",
		a.Name, b.Name, c.Name, d.Name, err, errB, errC, errD)

	text, err := shape.ParseJSON(shape.String().Trim().NotEmpty(), []byte(`" Shape "`))
	textWithContext, errContext := shape.ParseJSONContext(ctx, shape.String().Trim(), []byte(`" Context "`))
	items, errReader := shape.ParseJSONReader(shape.String().Trim().Slice(), strings.NewReader(`[" one "," two "]`))
	itemsWithContext, errReaderContext := shape.ParseJSONReaderContext(
		ctx,
		shape.Int().Positive().Slice(),
		strings.NewReader(`[1,2]`),
	)
	fmt.Printf("package parse: %q/%q %q %v errors=%v,%v,%v,%v\n",
		text, textWithContext, items, itemsWithContext,
		err, errContext, errReader, errReaderContext)

	var fromBytes Request
	var fromBytesContext Request
	var fromReader Request
	var fromReaderContext Request
	err = shape.BindJSON(&fromBytes, data, strict)
	errB = shape.BindJSONContext(ctx, &fromBytesContext, data, strict)
	errC = shape.BindJSONReader(&fromReader, strings.NewReader(string(data)), strict)
	errD = shape.BindJSONReaderContext(ctx, &fromReaderContext, strings.NewReader(string(data)), strict)
	fmt.Printf("bind: %q/%q/%q/%q errors=%v,%v,%v,%v\n",
		fromBytes.Name, fromBytesContext.Name, fromReader.Name, fromReaderContext.Name,
		err, errB, errC, errD)

	_, err = taggedSchema.ParseJSON([]byte(`{"name":"Pong","age":20,"extra":true}`), strict)
	fmt.Println("unknown field rejected:", err != nil)

	_, err = taggedSchema.ParseJSONReader(
		strings.NewReader(`{"name":"Pong","age":20}`),
		shape.JSONOptions{MaxBytes: 4},
	)
	fmt.Println("size limit identified:", errors.Is(err, shape.ErrJSONTooLarge))

	_, err = taggedSchema.ParseJSON([]byte(`{"name":"Pong","age":20} {}`))
	fmt.Println("second JSON value rejected:", err != nil)

	target := Request{Name: "keep", Age: 99}
	err = shape.BindJSON(&target, []byte(`{"name":"changed","age":10}`), strict)
	fmt.Printf("failed bind is atomic: %#v error=%v\n", target, err)
}
