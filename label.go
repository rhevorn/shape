package shape

import (
	"context"
	"errors"
)

// Label attaches a human-readable display name to schema. The name is copied
// into validation issues and interpolated by message catalogs as {{.Label}}.
// Prefer writing the name in the application's language; most apps only need
// one language, while issue templates still follow SetLanguage/WithLocale.
func Label[T any](name string, schema Schema[T]) Schema[T] {
	if schema == nil {
		panic("shape: labeled schema must not be nil")
	}
	return labeledSchema[T]{label: name, schema: schema}
}

type labeledSchema[T any] struct {
	label  string
	schema Schema[T]
}

func (s labeledSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

func (s labeledSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	parsed, err := s.schema.ParseContext(ctx, value)
	return parsed, withIssueLabel(ctx, err, s.label)
}

func (s labeledSchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	return buildJSONSchemaWithContext(s.schema, ctx)
}

func withIssueLabel(ctx context.Context, err error, label string) error {
	if err == nil || label == "" {
		return err
	}
	var validation *ValidationError
	if !errors.As(err, &validation) || validation == nil {
		return err
	}
	issues := make([]Issue, len(validation.Issues))
	for index, issue := range validation.Issues {
		issues[index] = issue
		if issues[index].Label == "" {
			issues[index].Label = label
		}
	}
	return validationIssues(ctx, issues)
}

func applyIssueLabel(issues []Issue, label string) []Issue {
	if label == "" || len(issues) == 0 {
		return issues
	}
	result := make([]Issue, len(issues))
	for index, issue := range issues {
		result[index] = issue
		if result[index].Label == "" {
			result[index].Label = label
		}
	}
	return result
}
