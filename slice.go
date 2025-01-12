package goshape

import (
	"context"
	"reflect"
)

// SliceSchema parses []any by applying a typed element schema to every item.
type SliceSchema[T any] struct {
	element     Schema[T]
	rules       []lengthRule
	refinements []refinement[[]T]
	constraints []map[string]any
	unique      bool
}

// Slice returns a schema for a slice of element values.
func Slice[T any](element Schema[T]) SliceSchema[T] {
	if element == nil {
		panic("goshape: slice element schema must not be nil")
	}
	return SliceSchema[T]{element: element}
}

// Min requires at least n elements.
func (s SliceSchema[T]) Min(n int) SliceSchema[T] {
	s.rules = appendCopy(s.rules, minLengthRule("Slice", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"minItems": n})
	return s
}

// Max allows at most n elements.
func (s SliceSchema[T]) Max(n int) SliceSchema[T] {
	s.rules = appendCopy(s.rules, maxLengthRule("Slice", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"maxItems": n})
	return s
}

// NonEmpty requires at least one element.
func (s SliceSchema[T]) NonEmpty() SliceSchema[T] { return s.Min(1) }

// Unique requires every pair of parsed elements to be deeply unequal.
func (s SliceSchema[T]) Unique() SliceSchema[T] {
	s.unique = true
	s.constraints = appendCopy(s.constraints, map[string]any{"uniqueItems": true})
	return s
}

func (s SliceSchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	items, err := buildJSONSchemaWithContext(s.element, ctx)
	if err != nil {
		return nil, err
	}
	document := map[string]any{"type": "array", "items": items}
	applyConstraints(document, s.constraints)
	return document, nil
}

// Refine adds custom validation after every element has parsed successfully.
func (s SliceSchema[T]) Refine(fn func([]T) error) SliceSchema[T] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[[]T].
func (s SliceSchema[T]) Parse(value any) ([]T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[[]T].
func (s SliceSchema[T]) ParseContext(ctx context.Context, value any) ([]T, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	input, ok := value.([]any)
	if !ok {
		if typed, typedOK := value.([]T); typedOK {
			input = make([]any, len(typed))
			for i := range typed {
				input[i] = typed[i]
			}
			ok = true
		}
	}
	if !ok {
		return nil, validationError(invalidType("[]any", value))
	}

	var issues []Issue
	for _, rule := range s.rules {
		if issue := rule(len(input)); issue != nil {
			issues = append(issues, *issue)
		}
	}
	if len(issues) != 0 {
		return nil, &ValidationError{Issues: issues}
	}

	result := make([]T, len(input))
	for index, item := range input {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		parsed, err := s.element.ParseContext(ctx, item)
		if err != nil {
			if contextErr := contextError(err, ctx); contextErr != nil {
				return nil, contextErr
			}
			issues = append(issues, prefixIssues(issuesFromError(err), IndexPath(index))...)
			continue
		}
		result[index] = parsed
	}
	if len(issues) != 0 {
		return nil, &ValidationError{Issues: issues}
	}
	if s.unique {
		for i := 0; i < len(result); i++ {
			for j := 0; j < i; j++ {
				if reflect.DeepEqual(result[i], result[j]) {
					return nil, validationError(Issue{
						Code:     CodeInvalidValue,
						Path:     Path{IndexPath(i)},
						Message:  "must contain unique items",
						Expected: "unique item",
						Received: result[i],
					})
				}
			}
		}
	}

	refinementIssues, err := runRefinements(ctx, result, s.refinements)
	if err != nil {
		return nil, err
	}
	if len(refinementIssues) != 0 {
		return nil, &ValidationError{Issues: refinementIssues}
	}
	return result, nil
}
