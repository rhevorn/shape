package main

import (
	"fmt"

	"github.com/rhevorn/shape"
)

type CreateUser struct {
	Name  string
	Email string
	Age   int
}

func main() {
	f := shape.Fields[CreateUser]()
	schema := shape.Object(
		f.Str("name", "姓名").Trim().Min(2).Max(50).Set(func(user *CreateUser, value string) {
			user.Name = value
		}),
		f.Email("email", "邮箱").Trim().Set(func(user *CreateUser, value string) {
			user.Email = value
		}),
		f.Int("age", "年龄").Min(18).Max(120).Set(func(user *CreateUser, value int) {
			user.Age = value
		}),
	).Strict()

	body := `{
		"name": " Pong ",
		"email": "pong@example.com",
		"age": 30
	}`

	user, err := shape.Parse(schema, body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", user)

	_, err = shape.Parse(schema, []byte(`{"name":"x","email":"bad","age":10}`))
	if err != nil {
		fmt.Println(err)
	}
}
