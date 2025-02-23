package shape_test

import (
	"testing"

	"github.com/rhevorn/shape"
)

func FuzzTaggedSchemaJSON(f *testing.F) {
	type Request struct {
		Name string `json:"name" shape:"trim,notempty,maxlength=50"`
		Age  int    `json:"age" shape:"min=0,max=150"`
	}
	schema := shape.Struct[Request]()
	f.Add([]byte(`{"name":" Pong ","age":20}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = schema.ParseJSON(data)
	})
}
