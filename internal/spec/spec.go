// Package spec contains immutable construction descriptors shared by shape's packages.
package spec

import (
	"context"
	"reflect"
)

type Option struct {
	name  string
	value any
}

func NewOption(name string, value any) Option { return Option{name: name, value: value} }
func OptionName(o Option) string              { return o.name }
func OptionValue(o Option) any                { return o.value }

type Rule struct {
	name  string
	args  []any
	type_ reflect.Type
	check func(context.Context, reflect.Value) error
}

func NewRule(name string, args ...any) Rule {
	return Rule{name: name, args: append([]any(nil), args...)}
}

func NewCustomRule(name string, target reflect.Type, check func(context.Context, reflect.Value) error) Rule {
	return Rule{name: name, type_: target, check: check}
}

func RuleName(r Rule) string                                      { return r.name }
func RuleTargetType(r Rule) reflect.Type                          { return r.type_ }
func RuleCheck(r Rule) func(context.Context, reflect.Value) error { return r.check }
func RuleArguments(r Rule) []any                                  { return append([]any(nil), r.args...) }
