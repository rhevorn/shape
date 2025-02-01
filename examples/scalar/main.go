package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rhevorn/shape"
)

func main() {
	// Single values parse directly — no map or struct required.
	name, err := shape.String().Trim().Min(2).Max(50).Parse("  Pong  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(name) // Pong

	age, err := shape.Int().Min(18).Max(120).Parse(30)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(age) // 30

	email, err := shape.String().Trim().Email().Parse("pong@example.com")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(email)

	// Transform changes the output type after a successful parse.
	port, err := shape.Transform(shape.String().Trim(), strconv.Atoi).Parse("8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(port) // 8080

	// Collections still validate each element with a scalar schema.
	tags, err := shape.Slice(shape.String().Trim().NonEmpty()).Min(1).Parse([]any{" go ", "shape"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(tags) // [go shape]

	// Failures return structured issues.
	_, err = shape.String().Min(3).Parse("x")
	var validation *shape.ValidationError
	if errors.As(err, &validation) {
		for _, issue := range validation.Issues {
			fmt.Println(issue.Code, issue.Message)
		}
	}
}
