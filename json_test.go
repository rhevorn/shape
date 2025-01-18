package goshape

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestParseJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Count int
		Ratio float64
	}
	schema := Object[payload](
		Field("count", Int(), func(value *payload, count int) { value.Count = count }),
		Field("ratio", Float64(), func(value *payload, ratio float64) { value.Ratio = ratio }),
	).Strict()

	got, err := ParseJSON(schema, []byte(`{"count": 1000e-2, "ratio": 1.25}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 10 || got.Ratio != 1.25 {
		t.Fatalf("ParseJSON = %#v", got)
	}
}

func TestParseJSONReader(t *testing.T) {
	t.Parallel()

	got, err := ParseJSONReader(String().Min(1), strings.NewReader(`"value"`))
	if err != nil || got != "value" {
		t.Fatalf("ParseJSONReader = %q, %v", got, err)
	}
	if _, err := ParseJSONReader(String(), nil); err == nil {
		t.Fatal("nil JSON reader was accepted")
	}
}

func TestParseJSONReaderLimit(t *testing.T) {
	t.Parallel()

	got, err := ParseJSONReaderLimit(String(), strings.NewReader(`"ok"`), 4)
	if err != nil || got != "ok" {
		t.Fatalf("exact limit = %q, %v", got, err)
	}
	if _, err := ParseJSONReaderLimit(String(), strings.NewReader(`"too large"`), 4); !errors.Is(err, ErrJSONTooLarge) {
		t.Fatalf("oversized error = %v", err)
	}
	if _, err := ParseJSONReaderLimit(String(), strings.NewReader(`null`), -1); err == nil {
		t.Fatal("negative limit was accepted")
	}
	if _, err := ParseJSONReaderLimit(String(), nil, 4); err == nil {
		t.Fatal("nil limited reader was accepted")
	}
}

func TestParseJSONErrors(t *testing.T) {
	t.Parallel()

	requireIssueCodes(t, parseJSONError(Int(), `1.5`), CodeInvalidNumber)
	for _, input := range []string{"", "{", "1 2", "true false"} {
		if err := parseJSONError(Int(), input); err == nil {
			t.Fatalf("expected JSON error for %q", input)
		} else {
			var validation *ValidationError
			if errors.As(err, &validation) {
				t.Fatalf("syntax/trailing error must not be ValidationError: %v", err)
			}
		}
	}
}

func TestJSONNumberRemainsStandardTypeForCustomSchema(t *testing.T) {
	t.Parallel()

	got, err := ParseJSON(jsonNumberSchema{}, []byte(`12345678901234567890`))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "12345678901234567890" {
		t.Fatalf("custom schema received %q", got)
	}
}

type jsonNumberSchema struct{}

func (jsonNumberSchema) Parse(value any) (json.Number, error) {
	return jsonNumberSchema{}.ParseContext(nil, value)
}

func (jsonNumberSchema) ParseContext(_ context.Context, value any) (json.Number, error) {
	number, ok := value.(json.Number)
	if !ok {
		return "", errors.New("not json.Number")
	}
	return number, nil
}

func parseJSONError[T any](schema Schema[T], input string) error {
	_, err := ParseJSON(schema, []byte(input))
	return err
}
