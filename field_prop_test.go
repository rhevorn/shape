package shape

import (
	"errors"
	"testing"
	"time"
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

func TestExtendedFieldFactorySmoke(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	duration := 90 * time.Minute

	type payload struct {
		Count    int64
		Ratio    float64
		StartsAt time.Time
		Timeout  time.Duration
	}

	f := Fields[payload]()
	schema := Object(
		f.Int64("count", "数量").Positive().Set(func(value *payload, count int64) {
			value.Count = count
		}),
		f.Float64("ratio").Gte(0).Lte(1).Set(func(value *payload, ratio float64) {
			value.Ratio = ratio
		}),
		f.Time("starts_at").Set(func(value *payload, startsAt time.Time) {
			value.StartsAt = startsAt
		}),
		f.Duration("timeout", "超时").Refine(func(d time.Duration) error {
			if d < time.Second {
				return errors.New("too short")
			}
			return nil
		}).Set(func(value *payload, timeout time.Duration) {
			value.Timeout = timeout
		}),
	)

	got, err := schema.Parse(map[string]any{
		"count":      int64(42),
		"ratio":      0.75,
		"starts_at":  now,
		"timeout":    duration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 42 || got.Ratio != 0.75 || !got.StartsAt.Equal(now) || got.Timeout != duration {
		t.Fatalf("got = %+v", got)
	}

	_, err = schema.Parse(map[string]any{
		"count":      int64(0),
		"ratio":      0.75,
		"starts_at":  now,
		"timeout":    duration,
	})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if len(validation.Issues) != 1 || validation.Issues[0].Path.String() != "count" {
		t.Fatalf("issues = %+v", validation.Issues)
	}
}
