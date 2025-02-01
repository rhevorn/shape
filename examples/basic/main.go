package main

import (
	"errors"
	"fmt"

	"github.com/rhevorn/shape"
)

type User struct {
	Name  string
	Email string
	Age   int
}

var userSchema = func() shape.ObjectSchema[User] {
	f := shape.Fields[User]()
	return shape.Object(
		f.Str("name").Trim().Min(2).Max(50).Set(func(user *User, value string) {
			user.Name = value
		}),
		f.Email("email").Trim().Set(func(user *User, value string) {
			user.Email = value
		}),
		f.Int("age").Min(18).Max(120).Set(func(user *User, value int) {
			user.Age = value
		}),
	).Strict()
}()

func main() {
	// Typical input is JSON text, not a Go map.
	body := []byte(`{
		"name": " Pong ",
		"email": "pong@example.com",
		"age": 30
	}`)

	user, err := shape.Parse(userSchema, body)
	if err != nil {
		var validation *shape.ValidationError
		if errors.As(err, &validation) {
			for _, issue := range validation.Issues {
				fmt.Printf("%s: %s\n", issue.Path, issue.Message)
			}
		}
		return
	}
	fmt.Printf("%+v\n", user)
}
