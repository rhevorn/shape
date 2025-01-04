package goshape

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestBool(t *testing.T) {
	t.Parallel()

	if got, err := Bool().Parse(true); err != nil || !got {
		t.Fatalf("Bool parse = %v, %v", got, err)
	}
	requireIssueCodes(t, parseError(Bool(), "true"), CodeInvalidType)
	requireIssueCodes(t, parseError(Bool().Refine(func(bool) error { return errors.New("no") }), true), CodeCustom)
}

func TestSlice(t *testing.T) {
	t.Parallel()

	schema := Slice(String().Min(2)).Min(1).Max(3)
	got, err := schema.Parse([]any{"ab", "cd"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"ab", "cd"}) {
		t.Fatalf("Slice parse = %#v", got)
	}

	issues := requireIssueCodes(t, parseError(schema, []any{"", "x"}), CodeTooSmall, CodeTooSmall)
	if issues[0].Path.String() != "[0]" || issues[1].Path.String() != "[1]" {
		t.Fatalf("slice paths = %q, %q", issues[0].Path, issues[1].Path)
	}
	requireIssueCodes(t, parseError(schema, []string{"ab"}), CodeInvalidType)
	requireIssueCodes(t, parseError(schema, []any{}), CodeTooSmall)
	requireIssueCodes(t, parseError(Slice(String()).Refine(func([]string) error {
		return NewIssue("slice_rule", "rejected")
	}), []any{"value"}), "slice_rule")
}

func TestMapDeterministicPaths(t *testing.T) {
	t.Parallel()

	schema := Map(Int().Min(1))
	issues := requireIssueCodes(t, parseError(schema, map[string]any{"z": 0, "a": "bad"}), CodeInvalidType, CodeTooSmall)
	if got := []string{issues[0].Path.String(), issues[1].Path.String()}; !reflect.DeepEqual(got, []string{"a", "z"}) {
		t.Fatalf("map paths = %v", got)
	}
	got, err := Map(String()).Parse(map[string]any{"key": "value"})
	if err != nil || got["key"] != "value" {
		t.Fatalf("Map parse = %#v, %v", got, err)
	}
	requireIssueCodes(t, parseError(Map(String()).Refine(func(map[string]string) error {
		return NewIssue("map_rule", "rejected")
	}), map[string]any{}), "map_rule")
}

func TestTransformAndGenericRefine(t *testing.T) {
	t.Parallel()

	port := Transform(String().Trim(), strconv.Atoi)
	if got, err := port.Parse(" 8080 "); err != nil || got != 8080 {
		t.Fatalf("transformed port = %d, %v", got, err)
	}
	requireIssueCodes(t, parseError(port, "wrong"), CodeTransformFailed)
	requireIssueCodes(t, parseError(port, 8080), CodeInvalidType)

	even := Refine(Int(), func(value int) error {
		if value%2 != 0 {
			return NewIssue("not_even", "must be even")
		}
		return nil
	})
	requireIssueCodes(t, parseError(even, 3), "not_even")
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Slice(String()).ParseContext(ctx, []any{"value"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ParseContext error = %v, want context.Canceled", err)
	}
}

func TestCompositionInvalidConfigurationPanics(t *testing.T) {
	t.Parallel()

	var nilStringSchema Schema[string]
	requirePanic(t, func() { Slice(nilStringSchema) })
	requirePanic(t, func() { Map(nilStringSchema) })
	requirePanic(t, func() { Refine(nilStringSchema, func(string) error { return nil }) })
	requirePanic(t, func() { Transform(nilStringSchema, func(string) (int, error) { return 0, nil }) })
	requirePanic(t, func() { Transform(String(), (func(string) (int, error))(nil)) })
	requirePanic(t, func() { Slice(String()).Min(-1) })
	requirePanic(t, func() { Slice(String()).Max(-1) })
}
