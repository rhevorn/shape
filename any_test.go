package shape

import "testing"

func TestAny(t *testing.T) {
	t.Parallel()

	got, err := Any().Parse([]any{1, "x", true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.([]any); !ok {
		t.Fatalf("Any = %#v", got)
	}

	object, err := Map(String(), Any()).Parse(map[string]any{"a": 1, "b": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if object["a"] != 1 || object["b"] != "x" {
		t.Fatalf("Map(String(), Any()) = %#v", object)
	}
}
