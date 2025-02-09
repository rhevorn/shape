package main

import (
	"fmt"

	"github.com/rhevorn/shape"
)

func main() {
	shape.SetLanguage("zh-CN")

	type User struct {
		Name  string
		Email string
		Age   int
	}

	f := shape.Fields[User]()
	schema := shape.Object(
		f.Str("name", "姓名").Trim().Min(2).Set(func(user *User, value string) {
			user.Name = value
		}),
		f.Email("email", "邮箱").Trim().Set(func(user *User, value string) {
			user.Email = value
		}),
		f.Int("age", "年龄").Min(18).Set(func(user *User, value int) {
			user.Age = value
		}),
	)

	_, err := shape.Parse(schema, `{
		"name": "x",
		"email": "plain",
		"age": 10
	}`)
	fmt.Println(err)

	_, err = shape.Label("昵称", shape.String().Min(3)).Parse("ab")
	fmt.Println(err)
}
