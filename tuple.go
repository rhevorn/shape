package shape

import "context"

// TupleElement is a strongly typed heterogeneous tuple position created with
// TupleItem.
type TupleElement[T any] interface {
	parseTupleItem(context.Context, *T, any, int) ([]Issue, error)
	buildTupleItemJSONSchema(*jsonSchemaBuildContext) (map[string]any, error)
}

// TupleItemDef connects one tuple position schema to a typed setter.
type TupleItemDef[T, V any] struct {
	schema Schema[V]
	setter func(*T, V)
}

// TupleItem creates one heterogeneous tuple position.
func TupleItem[T, V any](schema Schema[V], setter func(*T, V)) TupleItemDef[T, V] {
	if schema == nil {
		panic("shape: tuple item schema must not be nil")
	}
	if setter == nil {
		panic("shape: tuple item setter must not be nil")
	}
	return TupleItemDef[T, V]{schema: schema, setter: setter}
}

func (i TupleItemDef[T, V]) parseTupleItem(ctx context.Context, target *T, input any, index int) ([]Issue, error) {
	parsed, err := i.schema.ParseContext(ctx, input)
	if err != nil {
		if contextErr := contextError(err, ctx); contextErr != nil {
			return nil, contextErr
		}
		return prefixIssues(issuesFromError(err), IndexPath(index)), nil
	}
	i.setter(target, parsed)
	return nil, nil
}

func (i TupleItemDef[T, V]) buildTupleItemJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	return buildJSONSchemaWithContext(i.schema, ctx)
}

// TupleSchema parses a fixed-length heterogeneous JSON-style array into T.
type TupleSchema[T any] struct {
	items       []TupleElement[T]
	refinements []refinement[T]
}

// Tuple creates a fixed-length heterogeneous tuple schema.
func Tuple[T any](items ...TupleElement[T]) TupleSchema[T] {
	result := TupleSchema[T]{items: append([]TupleElement[T](nil), items...)}
	for _, item := range result.items {
		if item == nil {
			panic("shape: tuple item must not be nil")
		}
	}
	return result
}

// Refine adds validation after every tuple position parses successfully.
func (s TupleSchema[T]) Refine(fn func(T) error) TupleSchema[T] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[T].
func (s TupleSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[T].
func (s TupleSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	input, ok := value.([]any)
	if !ok {
		return zero, validationError(invalidType("[]any", value))
	}
	if len(input) != len(s.items) {
		code := CodeTooSmall
		if len(input) > len(s.items) {
			code = CodeTooBig
		}
		return zero, validationError(Issue{Code: code, Message: "must contain exactly the tuple length", Expected: len(s.items), Received: len(input)})
	}

	var candidate T
	var issues []Issue
	for index, item := range s.items {
		itemIssues, err := item.parseTupleItem(ctx, &candidate, input[index], index)
		if err != nil {
			return zero, err
		}
		var capped bool
		issues, capped = appendIssuesBounded(issues, itemIssues...)
		if capped {
			break
		}
	}
	if len(issues) != 0 {
		return zero, &ValidationError{Issues: issues}
	}
	refinementIssues, err := runRefinements(ctx, candidate, s.refinements)
	if err != nil {
		return zero, err
	}
	if len(refinementIssues) != 0 {
		return zero, &ValidationError{Issues: refinementIssues}
	}
	return candidate, nil
}

func (s TupleSchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	items := make([]any, len(s.items))
	for index, item := range s.items {
		document, err := item.buildTupleItemJSONSchema(ctx)
		if err != nil {
			return nil, err
		}
		items[index] = document
	}
	return map[string]any{
		"type":        "array",
		"prefixItems": items,
		"minItems":    len(items),
		"maxItems":    len(items),
		"items":       false,
	}, nil
}
