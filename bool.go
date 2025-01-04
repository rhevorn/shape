package goshape

import "context"

// BoolSchema parses strict bool values.
type BoolSchema struct {
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
	if !ok {
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
