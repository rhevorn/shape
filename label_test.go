package shape

import (
	"errors"
	"testing"
)

func TestLabelOnScalar(t *testing.T) {
	previous := Language()
	t.Cleanup(func() { SetLanguage(previous) })
	SetLanguage("en")

	_, err := Label("Display name", String().Min(3)).Parse("x")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if got := validation.Issues[0].Label; got != "Display name" {
		t.Fatalf("label = %q", got)
	}
	if got := validation.Issues[0].Message; got != "Display name must contain at least 3 characters" {
		t.Fatalf("message = %q", got)
	}
}

func TestFieldLabel(t *testing.T) {
	previous := Language()
	t.Cleanup(func() { SetLanguage(previous) })

	type user struct{ Name string }
	schema := Object[user](
		Field("name", String().Min(2), func(value *user, name string) {
			value.Name = name
		}).Label("姓名"),
	)

	SetLanguage("zh-CN")

	_, err := schema.Parse(map[string]any{"name": "x"})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if got := validation.Issues[0].Label; got != "姓名" {
		t.Fatalf("label = %q", got)
	}
	if got := validation.Issues[0].Message; got != "姓名至少需要 2 个字符" {
		t.Fatalf("message = %q", got)
	}

	_, err = schema.Parse(map[string]any{})
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if got := validation.Issues[0].Message; got != "姓名为必填项" {
		t.Fatalf("required message = %q", got)
	}
}
