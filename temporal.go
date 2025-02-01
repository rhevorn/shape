package shape

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// TimeSchema parses time.Time values.
type TimeSchema struct {
	coerce      bool
	layouts     []string
	refinements []refinement[time.Time]
}

// Time returns a strict time.Time schema.
func Time() TimeSchema { return TimeSchema{} }

// CoerceTime returns a time schema that also parses strings. It tries layouts
// in order; when none are supplied it uses time.RFC3339Nano.
func CoerceTime(layouts ...string) TimeSchema {
	if len(layouts) == 0 {
		layouts = []string{time.RFC3339Nano}
	}
	return TimeSchema{coerce: true, layouts: append([]string(nil), layouts...)}
}

// Refine adds custom time validation.
func (s TimeSchema) Refine(fn func(time.Time) error) TimeSchema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[time.Time].
func (s TimeSchema) Parse(value any) (time.Time, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[time.Time].
func (s TimeSchema) ParseContext(ctx context.Context, value any) (time.Time, error) {
	if err := checkContext(ctx); err != nil {
		return time.Time{}, err
	}
	parsed, ok := value.(time.Time)
	if !ok && s.coerce {
		if text, textOK := value.(string); textOK {
			text = strings.TrimSpace(text)
			for _, layout := range s.layouts {
				if candidate, err := time.Parse(layout, text); err == nil {
					parsed, ok = candidate, true
					break
				}
			}
		}
	}
	if !ok {
		if s.coerce {
			if _, isString := value.(string); isString {
				return time.Time{}, validationError(ctx, keyedIssue(CodeInvalidFormat, "invalid_format.time", s.layouts, value))
			}
		}
		return time.Time{}, validationError(ctx, invalidType("time.Time", value))
	}
	issues, err := runRefinements(ctx, parsed, s.refinements)
	if err != nil {
		return time.Time{}, err
	}
	if len(issues) != 0 {
		return time.Time{}, validationIssues(ctx, issues)
	}
	return parsed, nil
}

// DurationSchema parses time.Duration values.
type DurationSchema struct {
	coerce      bool
	refinements []refinement[time.Duration]
}

// Duration returns a strict time.Duration schema.
func Duration() DurationSchema { return DurationSchema{} }

// CoerceDuration returns a duration schema that also accepts strings supported
// by time.ParseDuration.
func CoerceDuration() DurationSchema { return DurationSchema{coerce: true} }

// Refine adds custom duration validation.
func (s DurationSchema) Refine(fn func(time.Duration) error) DurationSchema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[time.Duration].
func (s DurationSchema) Parse(value any) (time.Duration, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[time.Duration].
func (s DurationSchema) ParseContext(ctx context.Context, value any) (time.Duration, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(time.Duration)
	if !ok {
		if number, numberOK := value.(json.Number); numberOK {
			if integer, integerOK := parseJSONInteger(number.String(), 64); integerOK {
				parsed, ok = time.Duration(integer), true
			}
		}
	}
	if !ok && s.coerce {
		if text, textOK := value.(string); textOK {
			var err error
			parsed, err = time.ParseDuration(strings.TrimSpace(text))
			ok = err == nil
		}
	}
	if !ok {
		if s.coerce {
			if _, isString := value.(string); isString {
				return 0, validationError(ctx, keyedIssue(CodeInvalidFormat, "invalid_format.duration", "Go duration", value))
			}
		}
		return 0, validationError(ctx, invalidType("time.Duration", value))
	}
	issues, err := runRefinements(ctx, parsed, s.refinements)
	if err != nil {
		return 0, err
	}
	if len(issues) != 0 {
		return 0, validationIssues(ctx, issues)
	}
	return parsed, nil
}

func (s TimeSchema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	if !s.coerce {
		return nil, &UnsupportedSchemaError{Operation: "strict time.Time without a JSON representation; use CoerceTime"}
	}
	document := map[string]any{"type": "string", "x-shape-native-type": "time.Time"}
	if len(s.layouts) == 1 && (s.layouts[0] == time.RFC3339 || s.layouts[0] == time.RFC3339Nano) {
		document["format"] = "date-time"
	} else {
		document["x-shape-time-layouts"] = append([]string(nil), s.layouts...)
	}
	document["x-shape-coerce"] = true
	return document, nil
}

func (s DurationSchema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	if s.coerce {
		return map[string]any{"type": "string", "x-shape-format": "go-duration", "x-shape-coerce": true}, nil
	}
	return map[string]any{"type": "integer", "description": "duration in nanoseconds", "x-shape-native-type": "time.Duration"}, nil
}
