package shape_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
)

func ExampleString() {
	schema := shape.String().Trim().Min(3).Max(50)

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

	schema := shape.Object[User](
		shape.Field("name", shape.String().Trim().Min(2), func(user *User, value string) {
			user.Name = value
		}),
		shape.Field("email", shape.String().Trim().Email(), func(user *User, value string) {
			user.Email = value
		}),
		shape.Field("age", shape.Int().Min(18), func(user *User, value int) {
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
	_, err := shape.String().Min(3).Parse("x")
	var validation *shape.ValidationError
	if errors.As(err, &validation) {
		fmt.Println(validation.Issues[0].Code)
		fmt.Println(validation.Issues[0].Path.String())
	}
	// Output:
	// too_small
	// $
}

func ExampleWithLocale() {
	ctx := shape.WithLocale(context.Background(), "zh-CN")
	_, err := shape.String().Min(3).ParseContext(ctx, "x")
	var validation *shape.ValidationError
	if errors.As(err, &validation) {
		fmt.Println(validation.Issues[0].Message)
	}
	// Output: 至少需要 3 个字符
}

func ExampleTuple() {
	type Point struct {
		X int
		Y int
	}
	schema := shape.Tuple[Point](
		shape.TupleItem(shape.Int(), func(point *Point, value int) { point.X = value }),
		shape.TupleItem(shape.Int(), func(point *Point, value int) { point.Y = value }),
	)

	point, err := schema.Parse([]any{10, 20})
	if err != nil {
		panic(err)
	}
	fmt.Println(point.X, point.Y)
	// Output: 10 20
}

func ExampleMap() {
	schema := shape.Map(shape.String().ToLower(), shape.Int().Positive())

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
	var schema shape.Schema[Node]
	schema = shape.Lazy("Node", func() shape.Schema[Node] {
		return shape.Object[Node](
			shape.Field("value", shape.String().NonEmpty(), func(node *Node, value string) {
				node.Value = value
			}),
			shape.Field("children", shape.Slice(schema), func(node *Node, children []Node) {
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
	var schema shape.LazySchema[Node]
	schema = shape.Lazy("Node", func() shape.Schema[Node] {
		return shape.Object[Node](
			shape.Field("next", shape.Nullable[Node](schema), func(node *Node, next *Node) {
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
	schema := shape.Object[Config](
		shape.Field("labels", shape.Map(shape.String(), shape.String()), func(config *Config, labels map[string]string) {
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
	score, err := shape.Number[Score]().Min(0).Max(100).Parse(Score(95))
	if err != nil {
		panic(err)
	}
	fmt.Println(score)
	// Output: 95
}

func ExampleSlice() {
	values, err := shape.Slice(shape.String().Trim().NonEmpty()).Parse([]any{" one ", "two"})
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
	// Output: [one two]
}

func ExampleUnion() {
	contact := shape.Union[string](shape.String().Email(), shape.UUID())
	value, err := contact.Parse("pong@example.com")
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: pong@example.com
}

func ExampleOneOf() {
	schema := shape.OneOf[string](shape.String().Min(1), shape.String().Max(10))
	_, err := schema.Parse("overlap")
	var validation *shape.ValidationError
	if errors.As(err, &validation) {
		fmt.Println(validation.Issues[0].Code)
	}
	// Output: invalid_union
}

func ExampleNullable() {
	schema := shape.Nullable(shape.String())
	missing, _ := schema.Parse(nil)
	value, _ := schema.Parse("present")
	fmt.Println(missing == nil, *value)
	// Output: true present
}

func ExampleTransform() {
	schema := shape.Transform(shape.String().Trim(), strconv.Atoi)
	value, err := schema.Parse(" 8080 ")
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: 8080
}

func ExampleRefineContext() {
	schema := shape.RefineContext(shape.String(), func(ctx context.Context, value string) error {
		return ctx.Err()
	})
	value, err := schema.ParseContext(context.Background(), "available")
	fmt.Println(value, err)
	// Output: available <nil>
}

func ExampleParse() {
	type User struct {
		Name string
		Age  int
	}
	f := shape.Fields[User]()
	schema := shape.Object(
		f.Str("name").Trim().Min(1).Set(func(user *User, value string) {
			user.Name = value
		}),
		f.Int("age").Min(1).Set(func(user *User, value int) {
			user.Age = value
		}),
	)

	user, err := shape.Parse(schema, `{"name":" Pong ","age":30}`)
	if err != nil {
		panic(err)
	}
	fmt.Println(user.Name, user.Age)
	// Output: Pong 30
}

func ExampleMustStruct() {
	type User struct {
		Name  string `json:"name" shape:"trim,min=2,label='姓名'"`
		Email string `json:"email" shape:"trim,email"`
	}
	schema := shape.MustStruct[User]().Strict()
	user, err := shape.Parse(schema, `{"name":" Pong ","email":"pong@example.com"}`)
	if err != nil {
		panic(err)
	}
	fmt.Println(user.Name, user.Email)
	// Output: Pong pong@example.com
}

func ExampleBind() {
	type User struct {
		Name string `json:"name" shape:"trim,min=1"`
		Age  int    `json:"age" shape:"min=1"`
	}
	var user User
	err := shape.Bind(&user, `{"name":" Pong ","age":30}`)
	if err != nil {
		panic(err)
	}
	fmt.Println(user.Name, user.Age)
	// Output: Pong 30
}

func ExampleParseReaderLimit() {
	value, err := shape.ParseReaderLimit(shape.String(), strings.NewReader(`"value"`), 64)
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: value
}

func Example_jsonschema() {
	document, err := jsonschema.Export(shape.String().Min(2).Email())
	if err != nil {
		panic(err)
	}
	fmt.Println(document["type"], document["format"], document["minLength"], document["$schema"] != nil)
	// Output: string email 2 true
}

func ExampleAnnotate() {
	schema := shape.Annotate(shape.String()).Title("Display name").Example("Pong")
	fmt.Println(schema.Metadata().Title, schema.Metadata().Examples[0])
	// Output: Display name Pong
}
