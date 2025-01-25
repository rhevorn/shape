package goshape_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func ExampleLazySchema_MaxDepth() {
	type Node struct{ Next *Node }
	var schema goshape.LazySchema[Node]
	schema = goshape.Lazy("Node", func() goshape.Schema[Node] {
		return goshape.Object[Node](
			goshape.Field("next", goshape.Nullable[Node](schema), func(node *Node, next *Node) {
				node.Next = next
			}).Optional(),
		)
	}).MaxDepth(8)

	_, err := schema.Parse(map[string]any{})
	fmt.Println(err)
	// Output: <nil>
}

func ExampleFieldDef_DefaultFunc() {
	type Config struct{ Labels map[string]string }
	schema := goshape.Object[Config](
		goshape.Field("labels", goshape.Map(goshape.String()), func(config *Config, labels map[string]string) {
			config.Labels = labels
		}).DefaultFunc(func() map[string]string { return make(map[string]string) }),
	)

	first, _ := schema.Parse(map[string]any{})
	second, _ := schema.Parse(map[string]any{})
	first.Labels["request"] = "first"
	fmt.Println(len(first.Labels), len(second.Labels))
	// Output: 1 0
}

func ExampleNumber() {
	type Score int16
	score, err := goshape.Number[Score]().Min(0).Max(100).Parse(Score(95))
	if err != nil {
		panic(err)
	}
	fmt.Println(score)
	// Output: 95
}

func ExampleSlice() {
	values, err := goshape.Slice(goshape.String().Trim().NonEmpty()).Parse([]any{" one ", "two"})
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
	// Output: [one two]
}

func ExampleUnion() {
	contact := goshape.Union[string](goshape.String().Email(), goshape.UUID())
	value, err := contact.Parse("pong@example.com")
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: pong@example.com
}

func ExampleOneOf() {
	schema := goshape.OneOf[string](goshape.String().Min(1), goshape.String().Max(10))
	_, err := schema.Parse("overlap")
	var validation *goshape.ValidationError
	if errors.As(err, &validation) {
		fmt.Println(validation.Issues[0].Code)
	}
	// Output: invalid_union
}

func ExampleNullable() {
	schema := goshape.Nullable(goshape.String())
	missing, _ := schema.Parse(nil)
	value, _ := schema.Parse("present")
	fmt.Println(missing == nil, *value)
	// Output: true present
}

func ExampleTransform() {
	schema := goshape.Transform(goshape.String().Trim(), strconv.Atoi)
	value, err := schema.Parse(" 8080 ")
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: 8080
}

func ExampleRefineContext() {
	schema := goshape.RefineContext(goshape.String(), func(ctx context.Context, value string) error {
		return ctx.Err()
	})
	value, err := schema.ParseContext(context.Background(), "available")
	fmt.Println(value, err)
	// Output: available <nil>
}

func ExampleParseJSON() {
	values, err := goshape.ParseJSON(goshape.Slice(goshape.Int()), []byte(`[1,2,3]`))
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
	// Output: [1 2 3]
}

func ExampleParseJSONReaderLimit() {
	value, err := goshape.ParseJSONReaderLimit(goshape.String(), strings.NewReader(`"value"`), 64)
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: value
}

func ExampleJSONSchema() {
	document, err := goshape.JSONSchema(goshape.String().Min(2).Email())
	if err != nil {
		panic(err)
	}
	fmt.Println(document["type"], document["format"], document["minLength"])
	// Output: string email 2
}

func ExampleAnnotate() {
	schema := goshape.Annotate(goshape.String()).Title("Display name").Example("Pong")
	fmt.Println(schema.Metadata().Title, schema.Metadata().Examples[0])
	// Output: Display name Pong
}
