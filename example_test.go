package shape_test

import (
	"fmt"

	"github.com/rhevorn/shape"
)

func ExampleNew() {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	schema := shape.New[User](
		shape.Field("Name", shape.String().Trim().NotEmpty()),
		shape.Field("Age", shape.Int().Min(18)),
	)
	user, err := schema.ParseJSON([]byte(`{"name":" Pong ","age":20}`))

	fmt.Println(user.Name, user.Age, err)
	// Output: Pong 20 <nil>
}

func ExampleStruct() {
	type Request struct {
		Name string `json:"name" shape:"trim,notempty"`
	}

	schema := shape.Struct[Request]()
	request, err := schema.ParseJSON([]byte(`{"name":" Pong "}`))

	fmt.Println(request.Name, err)
	// Output: Pong <nil>
}

func ExampleBindJSON() {
	type Request struct {
		Name string `json:"name" shape:"trim,notempty"`
	}

	var request Request
	err := shape.BindJSON(&request, []byte(`{"name":" Pong "}`))

	fmt.Println(request.Name, err)
	// Output: Pong <nil>
}

func ExampleString() {
	name := shape.String().Trim().ToLower().NotEmpty()
	out, err := name.Transform(" Pong ")

	fmt.Println(out, err)
	fmt.Println(name.Validate(out))
	// Output:
	// pong <nil>
	// <nil>
}
