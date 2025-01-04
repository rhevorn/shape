package goshape

import "context"

// SliceSchema parses []any by applying a typed element schema to every item.
type SliceSchema[T any] struct {
	element     Schema[T]
	rules       []lengthRule
	refinements []refinement[[]T]
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
	return s
}

// Max allows at most n elements.
func (s SliceSchema[T]) Max(n int) SliceSchema[T] {
	s.rules = appendCopy(s.rules, maxLengthRule("Slice", n))
	return s
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

	refinementIssues, err := runRefinements(ctx, result, s.refinements)
	if err != nil {
		return nil, err
	}
	if len(refinementIssues) != 0 {
		return nil, &ValidationError{Issues: refinementIssues}
	}
	return result, nil
}
