package transform_test

import (
	"fmt"

	"github.com/rhevorn/shape/transform"
)

func ExampleString() {
	canonicalName := transform.String().Trim().ToLower()
	name, err := canonicalName.Transform(" Pong ")

	fmt.Println(name, err)
	// Output: pong <nil>
}
