// Package fieldmeta separates schema fields from their input names.
package fieldmeta

import (
	"context"
	"reflect"
	"strings"

	"github.com/rhevorn/shape/internal/jsonfields"
)

type Source uint8

const (
	Value Source = iota
	JSON
	Form
	Query
)

func (s Source) String() string {
	switch s {
	case JSON:
		return "JSON"
	case Form:
		return "form"
	case Query:
		return "query"
	default:
		return "value"
	}
}

// Names is immutable. An empty input name excludes the field from that input.
type Names struct {
	Default, JSON, Form, Query string
}

func Resolve(name string, tag reflect.StructTag) Names {
	jsonName, _ := jsonfields.Name(name, tag.Get("json"))
	if tag.Get("json") == "-" {
		jsonName = ""
	}
	n := Names{Default: jsonName, JSON: jsonName}
	if n.Default == "" {
		n.Default = name
	}
	n.Form = inputName(name, tag, "form")
	n.Query = inputName(name, tag, "query")
	return n
}

func inputName(goName string, tag reflect.StructTag, source string) string {
	name := tag.Get(source)
	if name == "-" {
		return ""
	}
	if name == "" {
		return goName
	}
	return name
}

// PathName falls back to the schema path for fields excluded from an input.
// Exclusion changes decoding, never the scope of transformation or validation.
func (n Names) PathName(source Source) string {
	if name := n.For(source); name != "" {
		return name
	}
	return n.Default
}

func (n Names) For(source Source) string {
	switch source {
	case JSON:
		return n.JSON
	case Form:
		return n.Form
	case Query:
		return n.Query
	default:
		return n.Default
	}
}

// ValidInputName reserves dots for nesting and brackets for error indexes.
// Form and query tags do not accept comma-separated options.
func ValidInputName(name string) bool {
	return name != "" && !strings.ContainsAny(name, ".[],")
}

type contextKey struct{}

func WithSource(ctx context.Context, source Source) context.Context {
	return context.WithValue(ctx, contextKey{}, source)
}

func FromContext(ctx context.Context) Source {
	source, _ := ctx.Value(contextKey{}).(Source)
	return source
}
