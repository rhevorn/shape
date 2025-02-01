package shape

import (
	"context"
	"reflect"
)

// DefaultMaxDeepUniqueItems bounds the quadratic fallback used when Go
// equality cannot preserve reflect.DeepEqual semantics.
const DefaultMaxDeepUniqueItems = 128

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
		panic("shape: slice element schema must not be nil")
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
		return nil, validationError(ctx, invalidType("[]any", value))
	}

	var issues []Issue
	for _, rule := range s.rules {
		if issue := rule(len(input)); issue != nil {
			issues, _ = appendIssuesBounded(issues, *issue)
		}
	}
	if len(issues) != 0 {
		return nil, validationIssues(ctx, issues)
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
			var capped bool
			issues, capped = appendIssuesBounded(issues, prefixIssues(issuesFromError(err), IndexPath(index))...)
			if capped {
				break
			}
			continue
		}
		result[index] = parsed
	}
	if len(issues) != 0 {
		return nil, validationIssues(ctx, issues)
	}
	if s.unique {
		deepItems := 0
		for index, value := range result {
			if index%256 == 0 {
				if err := checkContext(ctx); err != nil {
					return nil, err
				}
			}
			valueType := reflect.TypeOf(any(value))
			if valueType != nil && !deepEqualUsesEquality(valueType) {
				deepItems++
			}
		}
		if deepItems > DefaultMaxDeepUniqueItems {
			return nil, validationError(ctx, keyedIssue(CodeTooBig, "invalid_value.unique_deep_limit", DefaultMaxDeepUniqueItems, deepItems))
		}
		var seen map[any]struct{}
		for i := 0; i < len(result); i++ {
			if err := checkContext(ctx); err != nil {
				return nil, err
			}
			item := any(result[i])
			itemType := reflect.TypeOf(item)
			if itemType == nil || deepEqualUsesEquality(itemType) {
				if seen == nil {
					seen = make(map[any]struct{}, len(result)-deepItems)
				}
				if _, duplicate := seen[item]; duplicate {
					return nil, duplicateItemError(ctx, i, result[i])
				}
				seen[item] = struct{}{}
				continue
			}
			for j := 0; j < i; j++ {
				if j%64 == 0 {
					if err := checkContext(ctx); err != nil {
						return nil, err
					}
				}
				if reflect.TypeOf(any(result[j])) == itemType && reflect.DeepEqual(result[i], result[j]) {
					return nil, duplicateItemError(ctx, i, result[i])
				}
			}
		}
	}

	refinementIssues, err := runRefinements(ctx, result, s.refinements)
	if err != nil {
		return nil, err
	}
	if len(refinementIssues) != 0 {
		return nil, validationIssues(ctx, refinementIssues)
	}
	return result, nil
}

func duplicateItemError[T any](ctx context.Context, index int, value T) *ValidationError {
	issue := keyedIssue(CodeInvalidValue, "invalid_value.unique_items", "unique item", value)
	issue.Path = Path{IndexPath(index)}
	return validationError(ctx, issue)
}

// deepEqualUsesEquality reports types for which reflect.DeepEqual has the same
// semantics as Go equality. Pointer-bearing types deliberately use the
// fallback because DeepEqual also follows pointers.
func deepEqualUsesEquality(value reflect.Type) bool {
	switch value.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String, reflect.Chan, reflect.UnsafePointer:
		return true
	case reflect.Array:
		return deepEqualUsesEquality(value.Elem())
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if !deepEqualUsesEquality(value.Field(index).Type) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
