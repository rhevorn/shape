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

var userSchema = shape.Object[User](
	shape.Field("name", shape.String().Trim().Min(2).Max(50), func(user *User, value string) {
		user.Name = value
	}),
	shape.Field("email", shape.String().Trim().Email(), func(user *User, value string) {
		user.Email = value
	}),
	shape.Field("age", shape.Int().Min(18).Max(120), func(user *User, value int) {
		user.Age = value
	}),
).Strict()

func main() {
	user, err := userSchema.Parse(map[string]any{
		"name":  " Pong ",
		"email": "pong@example.com",
		"age":   30,
	})
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
