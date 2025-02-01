package shape

import (
	"errors"
	"testing"
)

func TestStrFieldSetWithOptionalLabel(t *testing.T) {
	previous := Language()
	t.Cleanup(func() { SetLanguage(previous) })
	SetLanguage("zh-CN")

	type user struct {
		Name  string
		Email string
		Age   int
	}

	f := Fields[user]()
	schema := Object(
		f.Str("name", "姓名").Trim().Min(2).Set(func(value *user, name string) {
			value.Name = name
		}),
		f.Email("email", "邮箱").Trim().Set(func(value *user, email string) {
			value.Email = email
		}),
		f.Int("age").Min(18).Set(func(value *user, age int) {
			value.Age = age
		}),
	)

	got, err := schema.Parse(map[string]any{
		"name":  " Pong ",
		"email": "pong@example.com",
		"age":   30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Pong" || got.Email != "pong@example.com" || got.Age != 30 {
		t.Fatalf("got = %+v", got)
	}

	_, err = schema.Parse(map[string]any{
		"name":  "x",
		"email": "plain",
		"age":   10,
	})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	messages := map[string]string{}
	for _, issue := range validation.Issues {
		messages[issue.Path.String()] = issue.Message
	}
	if got := messages["name"]; got != "姓名至少需要 2 个字符" {
		t.Fatalf("name message = %q", got)
	}
	if got := messages["email"]; got != "邮箱格式不正确" {
		t.Fatalf("email message = %q", got)
	}
	if got := messages["age"]; got != "必须大于或等于 18" {
		t.Fatalf("age message without label = %q", got)
	}
}
