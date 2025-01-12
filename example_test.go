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

func ExampleTuple() {
	type Point struct {
		X int
		Y int
	}
	schema := goshape.Tuple[Point](
		goshape.TupleItem(goshape.Int(), func(point *Point, value int) { point.X = value }),
		goshape.TupleItem(goshape.Int(), func(point *Point, value int) { point.Y = value }),
	)

	point, err := schema.Parse([]any{10, 20})
	if err != nil {
		panic(err)
	}
	fmt.Println(point.X, point.Y)
	// Output: 10 20
}

func ExampleRecord() {
	schema := goshape.Record(goshape.String().ToLower(), goshape.Int().Positive())

	values, err := schema.Parse(map[string]any{"ONE": 1})
	if err != nil {
		panic(err)
	}
	fmt.Println(values["one"])
	// Output: 1
}

func ExampleLazy() {
	type Node struct {
		Value    string
		Children []Node
	}
	var schema goshape.Schema[Node]
	schema = goshape.Lazy("Node", func() goshape.Schema[Node] {
		return goshape.Object[Node](
			goshape.Field("value", goshape.String().NonEmpty(), func(node *Node, value string) {
				node.Value = value
			}),
			goshape.Field("children", goshape.Slice(schema), func(node *Node, children []Node) {
				node.Children = children
			}).Default(nil),
		).Strict()
	})

	node, err := schema.Parse(map[string]any{
		"value": "root",
		"children": []any{
			map[string]any{"value": "leaf"},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(node.Value, node.Children[0].Value)
	// Output: root leaf
}
