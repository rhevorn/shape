package main

import (
	"errors"
	"fmt"

	"github.com/rhevorn/goshape"
)

type User struct {
	Name  string
	Email string
	Age   int
}

var userSchema = goshape.Object[User](
	goshape.Field("name", goshape.String().Trim().Min(2).Max(50), func(user *User, value string) {
		user.Name = value
	}),
	goshape.Field("email", goshape.String().Trim().Email(), func(user *User, value string) {
		user.Email = value
	}),
	goshape.Field("age", goshape.Int().Min(18).Max(120), func(user *User, value int) {
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
		var validation *goshape.ValidationError
		if errors.As(err, &validation) {
			for _, issue := range validation.Issues {
				fmt.Printf("%s: %s\n", issue.Path, issue.Message)
			}
		}
		return
	}
	fmt.Printf("%+v\n", user)
}
