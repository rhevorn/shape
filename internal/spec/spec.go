// Package spec contains immutable construction descriptors shared by shape's packages.
package spec

type Option struct {
	Name  string
	Value any
}

func NewOption(name string, value any) Option { return Option{Name: name, Value: value} }

type Rule struct {
	Name string
	Args []any
}

func NewRule(name string, args ...any) Rule {
	return Rule{Name: name, Args: append([]any(nil), args...)}
}
