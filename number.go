package goshape

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

type number interface {
	int | int64 | float64
}

type numberRule[N number] func(N) *Issue

// IntSchema parses and validates int values and decoder-native JSON numbers.
type IntSchema struct {
	rules       []numberRule[int]
	refinements []refinement[int]
}

// Int returns an int schema. It does not coerce strings or other Go number
// types.
func Int() IntSchema { return IntSchema{} }

// Min requires a value greater than or equal to bound.
func (s IntSchema) Min(bound int) IntSchema { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s IntSchema) Max(bound int) IntSchema { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s IntSchema) Gt(bound int) IntSchema {
	return s.withRule(lowerRule(bound, false))
}

// Gte requires a value greater than or equal to bound.
func (s IntSchema) Gte(bound int) IntSchema {
	return s.withRule(lowerRule(bound, true))
}

// Lt requires a value less than bound.
func (s IntSchema) Lt(bound int) IntSchema {
	return s.withRule(upperRule(bound, false))
}

// Lte requires a value less than or equal to bound.
func (s IntSchema) Lte(bound int) IntSchema {
	return s.withRule(upperRule(bound, true))
}

// Refine adds custom validation after built-in rules.
func (s IntSchema) Refine(fn func(int) error) IntSchema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[int].
func (s IntSchema) Parse(value any) (int, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[int].
func (s IntSchema) ParseContext(ctx context.Context, value any) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(int)
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed64, parseOK := parseJSONInteger(string(encoded), strconv.IntSize)
			if parseOK {
				parsed, ok = int(parsed64), true
			} else {
				return 0, invalidJSONIntegerError("int", encoded)
			}
		}
	}
	if !ok {
		return 0, validationError(invalidType("int", value))
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}
func (s IntSchema) withRule(rule numberRule[int]) IntSchema {
	s.rules = appendCopy(s.rules, rule)
	return s
}

// Int64Schema parses and validates int64 values and decoder-native JSON
// numbers.
type Int64Schema struct {
	rules       []numberRule[int64]
	refinements []refinement[int64]
}

// Int64 returns an int64 schema. It does not coerce strings or other Go number
// types.
func Int64() Int64Schema { return Int64Schema{} }

// Min requires a value greater than or equal to bound.
func (s Int64Schema) Min(bound int64) Int64Schema { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s Int64Schema) Max(bound int64) Int64Schema { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s Int64Schema) Gt(bound int64) Int64Schema {
	return s.withRule(lowerRule(bound, false))
}

// Gte requires a value greater than or equal to bound.
func (s Int64Schema) Gte(bound int64) Int64Schema {
	return s.withRule(lowerRule(bound, true))
}

// Lt requires a value less than bound.
func (s Int64Schema) Lt(bound int64) Int64Schema {
	return s.withRule(upperRule(bound, false))
}

// Lte requires a value less than or equal to bound.
func (s Int64Schema) Lte(bound int64) Int64Schema {
	return s.withRule(upperRule(bound, true))
}

// Refine adds custom validation after built-in rules.
func (s Int64Schema) Refine(fn func(int64) error) Int64Schema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[int64].
func (s Int64Schema) Parse(value any) (int64, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[int64].
func (s Int64Schema) ParseContext(ctx context.Context, value any) (int64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(int64)
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed, ok = parseJSONInteger(string(encoded), 64)
			if !ok {
				return 0, invalidJSONIntegerError("int64", encoded)
			}
		}
	}
	if !ok {
		return 0, validationError(invalidType("int64", value))
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}
func (s Int64Schema) withRule(rule numberRule[int64]) Int64Schema {
	s.rules = appendCopy(s.rules, rule)
	return s
}

// Float64Schema parses and validates finite float64 values and decoder-native
// JSON numbers.
type Float64Schema struct {
	rules       []numberRule[float64]
	refinements []refinement[float64]
}

// Float64 returns a float64 schema. It does not coerce strings or other Go
// number types.
func Float64() Float64Schema { return Float64Schema{} }

// Min requires a value greater than or equal to bound.
func (s Float64Schema) Min(bound float64) Float64Schema { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s Float64Schema) Max(bound float64) Float64Schema { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s Float64Schema) Gt(bound float64) Float64Schema {
	return s.withRule(lowerRule(bound, false))
}

// Gte requires a value greater than or equal to bound.
func (s Float64Schema) Gte(bound float64) Float64Schema {
	return s.withRule(lowerRule(bound, true))
}

// Lt requires a value less than bound.
func (s Float64Schema) Lt(bound float64) Float64Schema {
	return s.withRule(upperRule(bound, false))
}

// Lte requires a value less than or equal to bound.
func (s Float64Schema) Lte(bound float64) Float64Schema {
	return s.withRule(upperRule(bound, true))
}

// Refine adds custom validation after built-in rules.
func (s Float64Schema) Refine(fn func(float64) error) Float64Schema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[float64].
func (s Float64Schema) Parse(value any) (float64, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[float64].
func (s Float64Schema) ParseContext(ctx context.Context, value any) (float64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(float64)
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed, ok = parseJSONFloat(string(encoded))
			if !ok {
				return 0, validationError(Issue{
					Code:     CodeInvalidNumber,
					Message:  "must be a finite float64 number",
					Expected: "float64",
					Received: encoded.String(),
				})
			}
		}
	}
	if !ok {
		return 0, validationError(invalidType("float64", value))
	}
	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, validationError(Issue{
			Code:     CodeInvalidNumber,
			Message:  "must be a finite number",
			Expected: "finite float64",
			Received: parsed,
		})
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}
func (s Float64Schema) withRule(rule numberRule[float64]) Float64Schema {
	s.rules = appendCopy(s.rules, rule)
	return s
}

func lowerRule[N number](bound N, inclusive bool) numberRule[N] {
	return func(value N) *Issue {
		valid := value > bound
		comparison := "greater than"
		if inclusive {
			valid = value >= bound
			comparison = "greater than or equal to"
		}
		if valid {
			return nil
		}
		return &Issue{
			Code:     CodeTooSmall,
			Message:  fmt.Sprintf("must be %s %v", comparison, bound),
			Expected: bound,
			Received: value,
		}
	}
}

func upperRule[N number](bound N, inclusive bool) numberRule[N] {
	return func(value N) *Issue {
		valid := value < bound
		comparison := "less than"
		if inclusive {
			valid = value <= bound
			comparison = "less than or equal to"
		}
		if valid {
			return nil
		}
		return &Issue{
			Code:     CodeTooBig,
			Message:  fmt.Sprintf("must be %s %v", comparison, bound),
			Expected: bound,
			Received: value,
		}
	}
}

func parseNumber[N number](ctx context.Context, value N, rules []numberRule[N], refinements []refinement[N]) (N, error) {
	issues := make([]Issue, 0)
	for _, rule := range rules {
		if issue := rule(value); issue != nil {
			issues = append(issues, *issue)
		}
	}
	refinementIssues, err := runRefinements(ctx, value, refinements)
	if err != nil {
		var zero N
		return zero, err
	}
	issues = append(issues, refinementIssues...)
	if len(issues) != 0 {
		var zero N
		return zero, &ValidationError{Issues: issues}
	}
	return value, nil
}

func parseJSONInteger(value string, bits int) (int64, bool) {
	precision := uint(len(value)*4 + 64)
	parsed, _, err := big.ParseFloat(value, 10, precision, big.ToNearestEven)
	if err != nil {
		return 0, false
	}
	integer, accuracy := parsed.Int(nil)
	if accuracy != big.Exact || !integer.IsInt64() {
		return 0, false
	}
	result := integer.Int64()
	if bits == 32 && (result < math.MinInt32 || result > math.MaxInt32) {
		return 0, false
	}
	return result, true
}

func parseJSONFloat(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0, false
	}
	return parsed, true
}

func invalidJSONIntegerError(expected string, value json.Number) error {
	return validationError(Issue{
		Code:     CodeInvalidNumber,
		Message:  "must be an integer within the target type's range",
		Expected: expected,
		Received: value.String(),
	})
}
