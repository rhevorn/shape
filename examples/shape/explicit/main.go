package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhevorn/shape"
)

type Profile struct {
	Bio string `json:"bio"`
}

var profileSchema = shape.New[Profile](
	shape.String("Bio").Trim().MaxLength(200),
)

type User struct {
	Name     string         `json:"name"`
	Age      int            `json:"age"`
	Nickname *string        `json:"nickname"`
	Tags     []string       `json:"tags"`
	Scores   map[string]int `json:"scores"`
	Profile  Profile        `json:"profile"`
}

var userSchema = shape.New[User](
	shape.String("Name").Trim().NotEmpty().MaxLength(50),
	shape.Int("Age").Min(18).Max(120),
	shape.Pointer("Nickname", shape.String().Trim()),
	shape.Slice("Tags", shape.String().Trim().NotEmpty()).NotEmpty().Unique(),
	shape.Map("Scores", shape.String().Trim().NotEmpty(), shape.Int().NonNegative()),
	shape.Field("Profile", profileSchema),
).Apply(func(user User) (User, error) {
	user.Name = strings.Join(strings.Fields(user.Name), " ")
	return user, nil
}).Refine(func(user User) error {
	if user.Nickname != nil && *user.Nickname == user.Name {
		return errors.New("nickname must differ from name")
	}
	return nil
})

func main() {
	raw := User{Name: " Pong ", Age: 20, Tags: []string{" go ", "shape"}}
	user, err := userSchema.Transform(raw)
	if err == nil {
		err = userSchema.Validate(user)
	}
	fmt.Printf("typed: %#v error=%v\n", user, err)

	user, err = userSchema.ParseJSON([]byte(`{
		"name":" Pong ",
		"age":20,
		"nickname":" pp ",
		"tags":[" go ","shape"],
		"scores":{" docs ":10},
		"profile":{"bio":" library author "}
	}`))
	fmt.Printf("json: %#v error=%v\n", user, err)
}
