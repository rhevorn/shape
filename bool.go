package shape

import (
	"context"
	"encoding/json"
	"strings"
)

// BoolSchema parses strict bool values.
type BoolSchema struct {
	coerce      bool
	refinements []refinement[bool]
}

// Bool returns a strict boolean schema.
func Bool() BoolSchema {
	return BoolSchema{}
}

// Refine adds custom validation that runs after type validation.
func (s BoolSchema) Refine(fn func(bool) error) BoolSchema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[bool].
func (s BoolSchema) Parse(value any) (bool, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[bool].
func (s BoolSchema) ParseContext(ctx context.Context, value any) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	parsed, ok := value.(bool)
	if !ok && s.coerce {
		parsed, ok = coerceBoolValue(value)
	}
	if !ok {
		if s.coerce && isBoolCoercionCandidate(value) {
			return false, validationError(Issue{Code: CodeInvalidValue, Message: "cannot be converted to bool", Expected: "true, false, 1, or 0", Received: value})
		}
		return false, validationError(invalidType("bool", value))
	}
	issues, err := runRefinements(ctx, parsed, s.refinements)
	if err != nil {
		return false, err
	}
	if len(issues) != 0 {
		return false, &ValidationError{Issues: issues}
	}
	return parsed, nil
}

// CoerceBool returns a bool schema that accepts true/false and 1/0 scalar
// representations.
func CoerceBool() BoolSchema { return BoolSchema{coerce: true} }

func coerceBoolValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	case json.Number:
		if typed.String() == "1" {
			return true, true
		}
		if typed.String() == "0" {
			return false, true
		}
	case int:
		if typed == 1 {
			return true, true
		}
		if typed == 0 {
			return false, true
		}
	case int64:
		if typed == 1 {
			return true, true
		}
		if typed == 0 {
			return false, true
		}
	case float64:
		if typed == 1 {
			return true, true
		}
		if typed == 0 {
			return false, true
		}
	}
	return false, false
}

func isBoolCoercionCandidate(value any) bool {
	switch value.(type) {
	case string, json.Number, int, int64, float64:
		return true
	default:
		return false
	}
}

func (s BoolSchema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	document := map[string]any{"type": "boolean"}
	if s.coerce {
		document["x-shape-coerce"] = true
	}
	return document, nil
}
