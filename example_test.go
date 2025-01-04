package goshape_test

import (
	"errors"
	"fmt"

	"github.com/rhevorn/goshape"
)

func ExampleString() {
	schema := goshape.String().Trim().Min(3).Max(50)

	value, err := schema.Parse("  hello  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: hello
}

func ExampleObject() {
	type User struct {
		Name  string
		Email string
		Age   int
	}

	schema := goshape.Object[User](
		goshape.Field("name", goshape.String().Trim().Min(2), func(user *User, value string) {
			user.Name = value
		}),
		goshape.Field("email", goshape.String().Trim().Email(), func(user *User, value string) {
			user.Email = value
		}),
		goshape.Field("age", goshape.Int().Min(18), func(user *User, value int) {
			user.Age = value
		}),
	).Strict()

	user, err := schema.Parse(map[string]any{
		"name":  " Pong ",
		"email": "pong@example.com",
		"age":   30,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s, %s, %d\n", user.Name, user.Email, user.Age)
	// Output: Pong, pong@example.com, 30
}

func ExampleValidationError() {
	_, err := goshape.String().Min(3).Parse("x")
	var validation *goshape.ValidationError
	if errors.As(err, &validation) {
		fmt.Println(validation.Issues[0].Code)
		fmt.Println(validation.Issues[0].Path.String())
	}
	// Output:
	// too_small
	// $
}
