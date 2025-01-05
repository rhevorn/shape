package goshape

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestJSONSchemaObject(t *testing.T) {
	t.Parallel()

	type request struct {
		Name  string
		Role  string
		Tags  []string
		Score *int
	}
	schema := Object[request](
		Field("name", String().Min(2).Max(50).Description("Display name").Example("Pong"), func(value *request, field string) {
			value.Name = field
		}),
		Field("role", Enum("user", "admin"), func(value *request, field string) {
			value.Role = field
		}).Default("user"),
		Field("tags", Slice(String().NonEmpty()).Min(1).Max(5).Unique(), func(value *request, field []string) {
			value.Tags = field
		}).Optional(),
		Field("score", Nullable(Int().Min(0)), func(value *request, field *int) {
			value.Score = field
		}).Optional(),
	).Strict()

	document, err := JSONSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
	if document["$schema"] != jsonSchemaDraft202012 || document["type"] != "object" || document["additionalProperties"] != false {
		t.Fatalf("unexpected root schema: %#v", document)
	}
	if got, want := document["required"], []string{"name"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required = %#v, want %#v", got, want)
	}
	properties := document["properties"].(map[string]any)
	name := properties["name"].(map[string]any)
	if name["minLength"] != 2 || name["maxLength"] != 50 || name["description"] != "Display name" {
		t.Fatalf("name schema = %#v", name)
	}
	role := properties["role"].(map[string]any)
	if role["default"] != "user" || !reflect.DeepEqual(role["enum"], []string{"user", "admin"}) {
		t.Fatalf("role schema = %#v", role)
	}
	tags := properties["tags"].(map[string]any)
	if tags["minItems"] != 1 || tags["maxItems"] != 5 || tags["uniqueItems"] != true {
		t.Fatalf("tags schema = %#v", tags)
	}
	if _, err := document.Bytes(); err != nil {
		t.Fatal(err)
	}
}

func TestJSONSchemaMetadataWrapperAndFormats(t *testing.T) {
	t.Parallel()

	schema := Annotate(URL()).
		Title("Homepage").
		Description("Absolute homepage URL").
		Example("https://example.com").
		DefaultValue("https://example.com").
		Deprecated()
	document, err := JSONSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
	if document["format"] != "uri" || document["title"] != "Homepage" || document["deprecated"] != true {
		t.Fatalf("annotated schema = %#v", document)
	}
	if document["default"] != "https://example.com" {
		t.Fatalf("default metadata = %#v", document["default"])
	}
	metadata := schema.Metadata()
	metadata.Examples[0] = "mutated"
	if schema.Metadata().Examples[0] != "https://example.com" {
		t.Fatal("Metadata returned aliased example storage")
	}
}

func TestJSONSchemaUnionMapTemporalAndCoercion(t *testing.T) {
	t.Parallel()

	union, err := JSONSchema(Union[string](String().Email(), UUID()))
	if err != nil {
		t.Fatal(err)
	}
	if alternatives := union["anyOf"].([]any); len(alternatives) != 2 {
		t.Fatalf("union alternatives = %d", len(alternatives))
	}
	mapDocument, err := JSONSchema(Map(CoerceBool()))
	if err != nil {
		t.Fatal(err)
	}
	additional := mapDocument["additionalProperties"].(map[string]any)
	if additional["type"] != "boolean" || additional["x-goshape-coerce"] != true {
		t.Fatalf("map value schema = %#v", additional)
	}
	sizedMap, err := JSONSchema(Map(String()).Min(1).Max(3))
	if err != nil || sizedMap["minProperties"] != 1 || sizedMap["maxProperties"] != 3 {
		t.Fatalf("sized map schema = %#v, %v", sizedMap, err)
	}
	timeDocument, err := JSONSchema(CoerceTime())
	if err != nil || timeDocument["format"] != "date-time" {
		t.Fatalf("time schema = %#v, %v", timeDocument, err)
	}
}

func TestJSONSchemaUnsupportedOperations(t *testing.T) {
	t.Parallel()

	for _, run := range []func() error{
		func() error { _, err := JSONSchema(String().Refine(func(string) error { return nil })); return err },
		func() error {
			_, err := JSONSchema(Transform(String(), func(string) (int, error) { return 0, nil }))
			return err
		},
		func() error {
			_, err := JSONSchema(RefineContext(String(), func(context.Context, string) error { return nil }))
			return err
		},
		func() error { _, err := JSONSchema(Time()); return err },
	} {
		err := run()
		var unsupported *UnsupportedSchemaError
		if !errors.As(err, &unsupported) {
			t.Fatalf("expected UnsupportedSchemaError, got %T: %v", err, err)
		}
	}
}

func TestJSONSchemaMarshalsAsDraftDocument(t *testing.T) {
	t.Parallel()

	source := Int().Min(1).Max(10).OneOf(1, 2, 3)
	document, err := JSONSchema(source)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid JSON schema: %s", data)
	}
	document["enum"].([]int)[0] = 99
	again, err := JSONSchema(source)
	if err != nil {
		t.Fatal(err)
	}
	if again["enum"].([]int)[0] != 1 {
		t.Fatal("exported document aliased schema constraint storage")
	}
}
