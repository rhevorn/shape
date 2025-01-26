package shape

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestIntRules(t *testing.T) {
	t.Parallel()

	schema := Int().Min(1).Max(10).Gt(0).Gte(1).Lt(11).Lte(10)
	if got, err := schema.Parse(5); err != nil || got != 5 {
		t.Fatalf("Parse(5) = %d, %v", got, err)
	}
	requireIssueCodes(t, parseError(schema, 0), CodeTooSmall, CodeTooSmall, CodeTooSmall)
	requireIssueCodes(t, parseError(schema, 11), CodeTooBig, CodeTooBig, CodeTooBig)
	requireIssueCodes(t, parseError(schema, int64(5)), CodeInvalidType)
	requireIssueCodes(t, parseError(schema, "5"), CodeInvalidType)
}

func TestInt64AndFloat64(t *testing.T) {
	t.Parallel()

	if got, err := Int64().Min(2).Parse(int64(2)); err != nil || got != 2 {
		t.Fatalf("Int64 parse = %d, %v", got, err)
	}
	if got, err := Float64().Gt(1.5).Lte(2.5).Parse(2.0); err != nil || got != 2.0 {
		t.Fatalf("Float64 parse = %v, %v", got, err)
	}
	requireIssueCodes(t, parseError(Float64(), math.NaN()), CodeInvalidNumber)
	requireIssueCodes(t, parseError(Float64(), math.Inf(1)), CodeInvalidNumber)
}

func TestNumericJSONNumber(t *testing.T) {
	t.Parallel()

	if got, err := Int().Parse(json.Number("1e3")); err != nil || got != 1000 {
		t.Fatalf("Int JSON number = %d, %v", got, err)
	}
	if got, err := Int64().Parse(json.Number("9223372036854775807")); err != nil || got != math.MaxInt64 {
		t.Fatalf("Int64 max JSON number = %d, %v", got, err)
	}
	if got, err := Float64().Parse(json.Number("1.25")); err != nil || got != 1.25 {
		t.Fatalf("Float64 JSON number = %v, %v", got, err)
	}
	requireIssueCodes(t, parseError(Int(), json.Number("1.5")), CodeInvalidNumber)
	requireIssueCodes(t, parseError(Int64(), json.Number("9223372036854775808")), CodeInvalidNumber)
	requireIssueCodes(t, parseError(Float64(), json.Number("1e9999")), CodeInvalidNumber)
}

func TestIntegerExponentParsingIsBoundedAndExact(t *testing.T) {
	t.Parallel()

	accepted := map[string]int64{
		"1e3":       1000,
		"1000e-2":   10,
		"1.2300e2":  123,
		"-0e999999": 0,
		"-9.22e2":   -922,
	}
	for input, want := range accepted {
		got, err := Int64().Parse(json.Number(input))
		if err != nil || got != want {
			t.Fatalf("Int64(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
	for _, input := range []string{"1e600000000", "1e-600000000", "1.2e0", "9223372036854775808", "--1"} {
		requireIssueCodes(t, parseError(Int64(), json.Number(input)), CodeInvalidNumber)
	}
	if got, err := CoerceInt64().Parse("+42"); err != nil || got != 42 {
		t.Fatalf("CoerceInt64(+42) = %d, %v", got, err)
	}
}

func TestNumberRefine(t *testing.T) {
	t.Parallel()

	schema := Int().Refine(func(value int) error {
		if value%2 != 0 {
			return errors.New("must be even")
		}
		return nil
	})
	if _, err := schema.Parse(2); err != nil {
		t.Fatal(err)
	}
	requireIssueCodes(t, parseError(schema, 3), CodeCustom)
}
