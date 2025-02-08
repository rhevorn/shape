package shape

import (
	"reflect"
	"regexp"
	"testing"
)

type coordinate struct {
	X     int
	Y     int
	Label string
}

func coordinateSchema() TupleSchema[coordinate] {
	return Tuple[coordinate](
		TupleItem(Int(), func(value *coordinate, item int) { value.X = item }),
		TupleItem(Int(), func(value *coordinate, item int) { value.Y = item }),
		TupleItem(String().NonEmpty(), func(value *coordinate, item string) { value.Label = item }),
	)
}

func TestTuple(t *testing.T) {
	t.Parallel()

	got, err := coordinateSchema().Parse([]any{10, 20, "home"})
	if err != nil || got != (coordinate{X: 10, Y: 20, Label: "home"}) {
		t.Fatalf("Tuple parse = %#v, %v", got, err)
	}
	issues := requireIssueCodes(t, parseError(coordinateSchema(), []any{"x", 20, ""}), CodeInvalidType, CodeTooSmall)
	if paths := []string{issues[0].Path.String(), issues[1].Path.String()}; !reflect.DeepEqual(paths, []string{"[0]", "[2]"}) {
		t.Fatalf("tuple paths = %v", paths)
	}
	requireIssueCodes(t, parseError(coordinateSchema(), []any{1, 2}), CodeTooSmall)
}

func TestTupleJSONSchema(t *testing.T) {
	t.Parallel()

	document, err := ExportDocument(coordinateSchema())
	if err != nil {
		t.Fatal(err)
	}
	if document["minItems"] != 3 || document["maxItems"] != 3 || document["items"] != false {
		t.Fatalf("tuple schema = %#v", document)
	}
	if items := document["prefixItems"].([]any); len(items) != 3 {
		t.Fatalf("prefixItems = %#v", items)
	}
}

func TestMapKeyed(t *testing.T) {
	t.Parallel()

	schema := Map(String().ToLower().Pattern(regexp.MustCompile(`^[a-z]+$`)), Int().Positive()).NonEmpty()
	got, err := schema.Parse(map[string]any{"One": 1, "two": 2})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]int{"one": 1, "two": 2}) {
		t.Fatalf("Map parse = %#v", got)
	}
	typed, err := schema.Parse(map[string]int{"THREE": 3})
	if err != nil || !reflect.DeepEqual(typed, map[string]int{"three": 3}) {
		t.Fatalf("Map typed parse = %#v, %v", typed, err)
	}
	issues := requireIssueCodes(t, parseError(schema, map[string]any{"bad-key": 1, "ok": 0}), CodeInvalidFormat, CodeTooSmall)
	if paths := []string{issues[0].Path.String(), issues[1].Path.String()}; !reflect.DeepEqual(paths, []string{`["bad-key"]`, "ok"}) {
		t.Fatalf("record paths = %v", paths)
	}
	requireIssueCodes(t, parseError(schema, map[string]any{"A": 1, "a": 2}), CodeInvalidValue)
}

func TestMapKeyedJSONSchema(t *testing.T) {
	t.Parallel()

	schema := Map(String().Pattern(regexp.MustCompile(`^[a-z]+$`)), Bool()).Min(1).Max(5)
	document, err := ExportDocument(schema)
	if err != nil {
		t.Fatal(err)
	}
	if document["minProperties"] != 1 || document["maxProperties"] != 5 {
		t.Fatalf("record schema = %#v", document)
	}
	if document["propertyNames"].(map[string]any)["pattern"] != "^[a-z]+$" {
		t.Fatalf("propertyNames = %#v", document["propertyNames"])
	}
}
