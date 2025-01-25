package goshape

import (
	"errors"
	"reflect"
	"testing"
)

type testUser struct {
	Name     string
	Email    string
	Age      int
	Nickname string
}

func testUserSchema() ObjectSchema[testUser] {
	return Object[testUser](
		Field("name", String().Trim().Min(2).Max(50), func(user *testUser, value string) {
			user.Name = value
		}),
		Field("email", String().Trim().Email(), func(user *testUser, value string) {
			user.Email = value
		}),
		Field("age", Int().Min(18).Max(120), func(user *testUser, value int) {
			user.Age = value
		}),
		Field("nickname", String(), func(user *testUser, value string) {
			user.Nickname = value
		}).Optional(),
	)
}

func TestObjectDefinitionOfDone(t *testing.T) {
	t.Parallel()

	schema := testUserSchema().Strict()
	got, err := schema.Parse(map[string]any{
		"name":  " Pong ",
		"email": "pong@example.com",
		"age":   30,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := testUser{Name: "Pong", Email: "pong@example.com", Age: 30}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Object parse = %#v, want %#v", got, want)
	}
}

func TestObjectAggregatesFieldsAndUnknownKeys(t *testing.T) {
	t.Parallel()

	_, err := testUserSchema().Strict().Parse(map[string]any{
		"name":      "x",
		"email":     "bad",
		"age":       17,
		"z_unknown": true,
		"a_unknown": true,
	})
	issues := requireIssueCodes(t, err,
		CodeTooSmall,
		CodeInvalidEmail,
		CodeTooSmall,
		CodeUnknownField,
		CodeUnknownField,
	)
	gotPaths := make([]string, len(issues))
	for i, issue := range issues {
		gotPaths[i] = issue.Path.String()
	}
	wantPaths := []string{"name", "email", "age", "a_unknown", "z_unknown"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("issue paths = %v, want %v", gotPaths, wantPaths)
	}
}

func TestObjectMissingOptionalDefaultAndNil(t *testing.T) {
	t.Parallel()

	type settings struct {
		Required string
		Optional string
		Role     string
	}
	schema := Object[settings](
		Field("required", String(), func(value *settings, field string) { value.Required = field }),
		Field("optional", String(), func(value *settings, field string) { value.Optional = field }).Optional(),
		Field("role", String(), func(value *settings, field string) { value.Role = field }).Default("user"),
	)

	got, err := schema.Parse(map[string]any{"required": "yes"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (settings{Required: "yes", Role: "user"}); got != want {
		t.Fatalf("Object defaults = %#v, want %#v", got, want)
	}
	requireIssueCodes(t, parseError(schema, map[string]any{}), CodeRequired)
	issues := requireIssueCodes(t, parseError(schema, map[string]any{"required": "yes", "optional": nil}), CodeInvalidType)
	if issues[0].Path.String() != "optional" {
		t.Fatalf("nil optional path = %q", issues[0].Path)
	}
}

func TestObjectMutableDefaultsRequireFactory(t *testing.T) {
	t.Parallel()

	type settings struct{ Labels map[string]string }
	requirePanic(t, func() {
		_ = Field("labels", Map(String()), func(value *settings, labels map[string]string) {
			value.Labels = labels
		}).Default(map[string]string{"origin": "shared"})
	})

	schema := Object[settings](
		Field("labels", Map(String()), func(value *settings, labels map[string]string) {
			value.Labels = labels
		}).DefaultFunc(func() map[string]string {
			return map[string]string{"origin": "fresh"}
		}),
	)
	first, err := schema.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	first.Labels["request"] = "one"
	second, err := schema.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if _, shared := second.Labels["request"]; shared {
		t.Fatal("DefaultFunc reused mutable state")
	}
	var unsupported *UnsupportedSchemaError
	if _, err := JSONSchema(schema); !errors.As(err, &unsupported) {
		t.Fatalf("dynamic default export error = %T, %v", err, err)
	}
}

func TestObjectStripStrictAndImmutability(t *testing.T) {
	t.Parallel()

	base := testUserSchema()
	input := map[string]any{"name": "Pong", "email": "pong@example.com", "age": 30, "extra": true}
	if _, err := base.Parse(input); err != nil {
		t.Fatalf("strip mode rejected unknown field: %v", err)
	}
	if _, err := base.Strict().Strip().Parse(input); err != nil {
		t.Fatalf("explicit Strip rejected unknown field: %v", err)
	}
	requireIssueCodes(t, parseError(base.Strict(), input), CodeUnknownField)
	if _, err := base.Parse(input); err != nil {
		t.Fatalf("Strict mutated base schema: %v", err)
	}
}

func TestObjectRefine(t *testing.T) {
	t.Parallel()

	type interval struct{ Start, End int }
	called := false
	schema := Object[interval](
		Field("start", Int(), func(value *interval, field int) { value.Start = field }),
		Field("end", Int(), func(value *interval, field int) { value.End = field }),
	).Refine(func(value interval) error {
		called = true
		if value.Start > value.End {
			return NewIssue("invalid_range", "start must not exceed end")
		}
		return nil
	})

	requireIssueCodes(t, parseError(schema, map[string]any{"start": 2, "end": 1}), "invalid_range")
	called = false
	requireIssueCodes(t, parseError(schema, map[string]any{"start": "bad", "end": 1}), CodeInvalidType)
	if called {
		t.Fatal("object refinement ran on an invalid partial object")
	}
}

func TestNestedObjectAndSlicePath(t *testing.T) {
	t.Parallel()

	type address struct{ ZIP string }
	type person struct{ Address address }
	addressSchema := Object[address](
		Field("zip", String().Len(5), func(value *address, zip string) { value.ZIP = zip }),
	)
	personSchema := Object[person](
		Field("address", addressSchema, func(value *person, address address) { value.Address = address }),
	)
	schema := Object[struct{ Users []person }](
		Field("users", Slice(personSchema), func(value *struct{ Users []person }, users []person) { value.Users = users }),
	)

	issues := requireIssueCodes(t, parseError(schema, map[string]any{
		"users": []any{
			map[string]any{"address": map[string]any{"zip": "x"}},
		},
	}), CodeTooSmall)
	if got, want := issues[0].Path.String(), "users[0].address.zip"; got != want {
		t.Fatalf("nested path = %q, want %q", got, want)
	}
}

func TestObjectInvalidConfigurationPanics(t *testing.T) {
	t.Parallel()

	type target struct{ Value string }
	field := Field("value", String(), func(value *target, field string) { value.Value = field })
	requirePanic(t, func() { Object[target](field, field) })
	requirePanic(t, func() { Field("", String(), func(*target, string) {}) })
	requirePanic(t, func() { Field("value", String(), (func(*target, string))(nil)) })
	requirePanic(t, func() { Object[target]((ObjectField[target])(nil)) })
}

func TestObjectRefinementPlainError(t *testing.T) {
	t.Parallel()

	schema := Object[struct{}]().Refine(func(struct{}) error { return errors.New("no") })
	requireIssueCodes(t, parseError(schema, map[string]any{}), CodeCustom)
}
