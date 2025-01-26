package shape

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestExtendedStringRulesAndFormats(t *testing.T) {
	t.Parallel()

	got, err := String().Trim().ToLower().StartsWith("go").EndsWith("shape").Contains("sha").Parse("  GoShape  ")
	if err != nil || got != "goshape" {
		t.Fatalf("normalized string = %q, %v", got, err)
	}
	requireIssueCodes(t, parseError(String().NonEmpty(), ""), CodeTooSmall)
	requireIssueCodes(t, parseError(String().StartsWith("x"), "abc"), CodeInvalidString)
	requireIssueCodes(t, parseError(String().EndsWith("x"), "abc"), CodeInvalidString)
	requireIssueCodes(t, parseError(String().Contains("x"), "abc"), CodeInvalidString)

	if _, err := URL().Parse("https://example.com/path"); err != nil {
		t.Fatal(err)
	}
	if _, err := UUID().Parse("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}
	if _, err := IP().Parse("2001:db8::1"); err != nil {
		t.Fatal(err)
	}
	requireIssueCodes(t, parseError(URL(), "/relative"), CodeInvalidURL)
	requireIssueCodes(t, parseError(UUID(), "not-a-uuid"), CodeInvalidUUID)
	requireIssueCodes(t, parseError(IP(), "999.1.1.1"), CodeInvalidIP)
}

func TestCoercion(t *testing.T) {
	t.Parallel()

	if got, err := CoerceString().Parse(42); err != nil || got != "42" {
		t.Fatalf("CoerceString = %q, %v", got, err)
	}
	if got, err := CoerceInt().Parse(" 42 "); err != nil || got != 42 {
		t.Fatalf("CoerceInt = %d, %v", got, err)
	}
	if got, err := CoerceInt64().Parse(uint32(42)); err != nil || got != 42 {
		t.Fatalf("CoerceInt64 = %d, %v", got, err)
	}
	if got, err := CoerceFloat().Parse("1.25"); err != nil || got != 1.25 {
		t.Fatalf("CoerceFloat = %v, %v", got, err)
	}
	if got, err := CoerceBool().Parse("TRUE"); err != nil || !got {
		t.Fatalf("CoerceBool = %v, %v", got, err)
	}
	if got, err := CoerceBool().Parse(json.Number("0")); err != nil || got {
		t.Fatalf("CoerceBool JSON zero = %v, %v", got, err)
	}

	requireIssueCodes(t, parseError(Int(), "42"), CodeInvalidType)
	requireIssueCodes(t, parseError(CoerceInt(), 1.5), CodeInvalidNumber)
	requireIssueCodes(t, parseError(CoerceString(), math.Inf(1)), CodeInvalidType)
	requireIssueCodes(t, parseError(CoerceBool(), "yes"), CodeInvalidValue)
}

func TestNumberConvenienceRules(t *testing.T) {
	t.Parallel()

	if _, err := Int().Positive().OneOf(1, 2).Parse(2); err != nil {
		t.Fatal(err)
	}
	requireIssueCodes(t, parseError(Int().Positive(), 0), CodeTooSmall)
	requireIssueCodes(t, parseError(Int64().Negative(), int64(0)), CodeTooBig)
	requireIssueCodes(t, parseError(Float64().NonNegative(), -1.0), CodeTooSmall)
	requireIssueCodes(t, parseError(Int().OneOf(1, 2), 3), CodeInvalidEnum)
	requirePanic(t, func() { Int().OneOf() })
}

func TestTypedCollectionsAndUnique(t *testing.T) {
	t.Parallel()

	got, err := Slice(String().NonEmpty()).Unique().Parse([]string{"a", "b"})
	if err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("typed slice = %#v, %v", got, err)
	}
	issues := requireIssueCodes(t, parseError(Slice(String()).Unique(), []string{"a", "a"}), CodeInvalidValue)
	if issues[0].Path.String() != "[1]" {
		t.Fatalf("duplicate path = %q", issues[0].Path)
	}
	mapValue, err := Map(Int()).Parse(map[string]int{"one": 1})
	if err != nil || mapValue["one"] != 1 {
		t.Fatalf("typed map = %#v, %v", mapValue, err)
	}
	requireIssueCodes(t, parseError(Map(String()).NonEmpty(), map[string]any{}), CodeTooSmall)
}

func TestEnumLiteralAndUnion(t *testing.T) {
	t.Parallel()

	colors := Enum("red", "green", "blue")
	if got, err := colors.Parse("green"); err != nil || got != "green" {
		t.Fatalf("Enum = %q, %v", got, err)
	}
	requireIssueCodes(t, parseError(colors, "yellow"), CodeInvalidEnum)
	if got, err := ParseJSON(Enum(1, 2, 3), []byte(`2`)); err != nil || got != 2 {
		t.Fatalf("numeric JSON enum = %d, %v", got, err)
	}
	if got, err := ParseJSON(Literal(42), []byte(`42`)); err != nil || got != 42 {
		t.Fatalf("numeric JSON literal = %d, %v", got, err)
	}
	requireIssueCodes(t, parseError(Literal("fixed"), "other"), CodeInvalidValue)

	contact := OneOf[string](String().Email(), String().Pattern(regexp.MustCompile(`^user-[0-9]+$`)))
	if _, err := contact.Parse("user-42"); err != nil {
		t.Fatal(err)
	}
	requireIssueCodes(t, parseError(contact, "invalid"), CodeInvalidUnion)
	if _, err := Union[string](String().Len(1), String().Len(2)).Parse("ab"); err != nil {
		t.Fatal(err)
	}
	requireIssueCodes(t, parseError(
		OneOf[string](String().Min(1), String().Max(10)),
		"overlap",
	), CodeInvalidUnion)

	requirePanic(t, func() { Enum[int]() })
	requirePanic(t, func() { OneOf[string]() })
	requirePanic(t, func() { Union[string]() })

	type status string
	if got, err := ParseJSON(Enum(status("open"), status("closed")), []byte(`"open"`)); err != nil || got != status("open") {
		t.Fatalf("named string enum = %q, %v", got, err)
	}
}

func TestNullable(t *testing.T) {
	t.Parallel()

	schema := Nullable(String().Min(1))
	if got, err := schema.Parse(nil); err != nil || got != nil {
		t.Fatalf("nullable nil = %#v, %v", got, err)
	}
	got, err := schema.Parse("value")
	if err != nil || got == nil || *got != "value" {
		t.Fatalf("nullable value = %#v, %v", got, err)
	}
	requireIssueCodes(t, parseError(schema, ""), CodeTooSmall)
}

func TestTemporalSchemas(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	if got, err := Time().Parse(now); err != nil || !got.Equal(now) {
		t.Fatalf("Time = %v, %v", got, err)
	}
	if got, err := CoerceTime().Parse("2026-09-02T12:00:00Z"); err != nil || !got.Equal(now) {
		t.Fatalf("CoerceTime = %v, %v", got, err)
	}
	if got, err := Duration().Parse(2 * time.Second); err != nil || got != 2*time.Second {
		t.Fatalf("Duration = %v, %v", got, err)
	}
	if got, err := CoerceDuration().Parse("1h30m"); err != nil || got != 90*time.Minute {
		t.Fatalf("CoerceDuration = %v, %v", got, err)
	}
	requireIssueCodes(t, parseError(Time(), "2026-09-02T12:00:00Z"), CodeInvalidType)
	requireIssueCodes(t, parseError(CoerceTime(), "not-time"), CodeInvalidFormat)
}

func TestRefineContext(t *testing.T) {
	t.Parallel()

	type contextKey string
	schema := RefineContext(String(), func(ctx context.Context, value string) error {
		if ctx.Value(contextKey("allowed")) != value {
			return errors.New("not allowed")
		}
		return nil
	})
	ctx := context.WithValue(context.Background(), contextKey("allowed"), "pong")
	if _, err := schema.ParseContext(ctx, "pong"); err != nil {
		t.Fatal(err)
	}
	requireIssueCodes(t, parseContextError(schema, context.Background(), "pong"), CodeCustom)
}

func parseContextError[T any](schema Schema[T], ctx context.Context, value any) error {
	_, err := schema.ParseContext(ctx, value)
	return err
}
