package goshape

import (
	"errors"
	"regexp"
	"testing"
)

func TestStringParseAndRules(t *testing.T) {
	t.Parallel()

	schema := String().Trim().Min(2).Max(4).Pattern(regexp.MustCompile(`^你.*$`))
	got, err := schema.Parse("  你好  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "你好" {
		t.Fatalf("Parse() = %q, want %q", got, "你好")
	}

	_, err = String().Len(2).Parse("👍🏽")
	if err != nil {
		t.Fatalf("Len must count Unicode code points: %v", err)
	}
}

func TestStringCollectsRulesAndRefinements(t *testing.T) {
	t.Parallel()

	schema := String().Min(3).Email().Refine(func(string) error {
		return errors.New("business rule failed")
	})
	_, err := schema.Parse("x")
	requireIssueCodes(t, err, CodeTooSmall, CodeInvalidEmail, CodeCustom)
}

func TestStringEmail(t *testing.T) {
	t.Parallel()

	schema := String().Trim().Email()
	if got, err := schema.Parse(" hello@example.com "); err != nil || got != "hello@example.com" {
		t.Fatalf("valid email: got %q, err %v", got, err)
	}
	for _, input := range []string{"not-an-email", "Name <hello@example.com>", "hello @example.com"} {
		if _, err := schema.Parse(input); err == nil {
			t.Fatalf("expected %q to be rejected", input)
		}
	}
}

func TestStringIsStrictAndImmutable(t *testing.T) {
	t.Parallel()

	base := String().Min(1)
	short := base.Max(2)
	email := base.Email()

	if _, err := base.Parse("long value"); err != nil {
		t.Fatalf("derived schema mutated base: %v", err)
	}
	requireIssueCodes(t, parseError(short, "long value"), CodeTooBig)
	requireIssueCodes(t, parseError(email, "plain"), CodeInvalidEmail)
	requireIssueCodes(t, parseError(base, 12), CodeInvalidType)
}

func TestStringInvalidConfigurationPanics(t *testing.T) {
	t.Parallel()

	requirePanic(t, func() { String().Min(-1) })
	requirePanic(t, func() { String().Max(-1) })
	requirePanic(t, func() { String().Len(-1) })
	requirePanic(t, func() { String().Pattern(nil) })
	requirePanic(t, func() { String().Refine(nil) })
}

func parseError[T any](schema Schema[T], value any) error {
	_, err := schema.Parse(value)
	return err
}
