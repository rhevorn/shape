package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

import "github.com/rhevorn/shape"

type Address struct {
	City string `json:"city"`
}

var addressSchema = shape.New[Address](
	shape.Field("City", shape.String().Trim().NotEmpty()),
)

type Metadata struct {
	Source string `json:"source"`
}

type User struct {
	Name     string         `json:"name"`
	Age      int            `json:"age"`
	Active   bool           `json:"active"`
	Nickname *string        `json:"nickname"`
	Tags     []string       `json:"tags"`
	Scores   map[string]int `json:"scores"`
	Address  Address        `json:"address"`
	Metadata Metadata       `json:"metadata"`
}

var userSchema = shape.New[User](
	shape.Field("Name", shape.String().Trim().NotEmpty().MaxLength(50)),
	shape.Field("Age", shape.Int().Between(18, 120)),
	shape.Field("Active", shape.Bool()),
	shape.Field("Nickname", shape.Pointer(shape.String().Trim().NotEmpty())),
	shape.Field("Tags", shape.Slice(shape.String().Trim().NotEmpty()).NotEmpty().Unique()),
	shape.Field("Scores", shape.Map(shape.String().Trim().NotEmpty(), shape.Int().NonNegative())),
	shape.Field("Address", addressSchema),
	shape.Field("Metadata", shape.Value[Metadata]().Apply(func(value Metadata) (Metadata, error) {
		if value.Source == "" {
			value.Source = "api"
		}
		return value, nil
	})),
).Apply(
	func(user User) (User, error) {
		user.Name = strings.Join(strings.Fields(user.Name), " ")
		return user, nil
	},
).ApplyContext(
	func(ctx context.Context, user User) (User, error) {
		return user, ctx.Err()
	},
).Refine(
	func(user User) error {
		if user.Nickname != nil && *user.Nickname == user.Name {
			return errors.New("nickname must differ from name")
		}
		return nil
	},
).RefineContext(
	func(ctx context.Context, _ User) error { return ctx.Err() },
)

func main() {
	raw := User{
		Name: "  Pong  ", Age: 20, Active: true,
		Tags:    []string{" go ", " shape "},
		Scores:  map[string]int{" docs ": 10},
		Address: Address{City: " Shanghai "},
	}

	user, err := userSchema.Transform(raw)
	fmt.Printf("transform: name=%q tags=%q city=%q source=%q error=%v\n",
		user.Name, user.Tags, user.Address.City, user.Metadata.Source, err)
	fmt.Printf("input unchanged: name=%q first-tag=%q\n", raw.Name, raw.Tags[0])

	err = userSchema.Validate(user)
	fmt.Println("validate transformed:", err)

	err = userSchema.Validate(User{Name: "", Age: 15})
	fmt.Println("validate invalid:", err)

	parsed, err := userSchema.ParseJSON([]byte(`{
		"name":"  Shape User  ",
		"age":30,
		"active":true,
		"nickname":" friend ",
		"tags":[" go "," schema "],
		"scores":{" examples ":5},
		"address":{"city":" Hangzhou "},
		"metadata":{}
	}`))
	fmt.Printf("json: name=%q nickname=%q source=%q error=%v\n",
		parsed.Name, dereference(parsed.Nickname), parsed.Metadata.Source, err)

	first := userSchema.ValidateFirst(User{})
	fmt.Println("validate first:", first)
}

func dereference(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
