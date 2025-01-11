package goshape

import (
	"math"
	"testing"
)

type percentage uint8
type temperature float32
type sequence int64

func TestGenericNumberNamedTypes(t *testing.T) {
	t.Parallel()

	percent := Number[percentage]().Min(0).Max(100)
	if got, err := percent.Parse(percentage(75)); err != nil || got != 75 {
		t.Fatalf("named strict number = %v, %v", got, err)
	}
	if got, err := ParseJSON(percent, []byte(`75`)); err != nil || got != 75 {
		t.Fatalf("named JSON number = %v, %v", got, err)
	}
	requireIssueCodes(t, parseJSONError(percent, `256`), CodeInvalidNumber)
	requireIssueCodes(t, parseError(percent, uint8(75)), CodeInvalidType)

	if got, err := CoerceNumber[sequence]().Parse("42"); err != nil || got != 42 {
		t.Fatalf("coerced named integer = %v, %v", got, err)
	}
	if got, err := CoerceNumber[temperature]().Parse("1.25"); err != nil || got != temperature(1.25) {
		t.Fatalf("coerced named float = %v, %v", got, err)
	}
}

func TestGenericNumberUnsignedAndFloatBoundaries(t *testing.T) {
	t.Parallel()

	if got, err := ParseJSON(Number[uint64](), []byte(`18446744073709551615`)); err != nil || got != math.MaxUint64 {
		t.Fatalf("uint64 maximum = %v, %v", got, err)
	}
	requireIssueCodes(t, parseJSONError(Number[uint64](), `18446744073709551616`), CodeInvalidNumber)
	requireIssueCodes(t, parseJSONError(Number[uint64](), `-1`), CodeInvalidNumber)
	requireIssueCodes(t, parseJSONError(Number[float32](), `1e100`), CodeInvalidNumber)
}

func TestGenericNumberJSONSchema(t *testing.T) {
	t.Parallel()

	integer, err := JSONSchema(Number[percentage]().Min(1).Max(100))
	if err != nil || integer["type"] != "integer" || integer["minimum"] != percentage(1) {
		t.Fatalf("integer JSON Schema = %#v, %v", integer, err)
	}
	floating, err := JSONSchema(CoerceNumber[temperature]())
	if err != nil || floating["type"] != "number" || floating["x-goshape-coerce"] != true {
		t.Fatalf("float JSON Schema = %#v, %v", floating, err)
	}
}
