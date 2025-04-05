package transform

import "context"

// StringTransformer transforms strings with built-in normalization steps.
type StringTransformer struct{ ValueTransformer[string] }

// IfZero replaces an empty string with v.
func (t StringTransformer) IfZero(v string) StringTransformer {
	t.ValueTransformer = t.ValueTransformer.IfZero(v)
	return t
}

// Trim removes surrounding Unicode whitespace, or characters from one cutset.
func (t StringTransformer) Trim(chars ...string) StringTransformer {
	t.ValueTransformer = t.appendSteps(trim(trimBoth, chars))
	return t
}

// LTrim removes leading Unicode whitespace, or characters from one cutset.
func (t StringTransformer) LTrim(chars ...string) StringTransformer {
	t.ValueTransformer = t.appendSteps(trim(trimLeft, chars))
	return t
}

// RTrim removes trailing Unicode whitespace, or characters from one cutset.
func (t StringTransformer) RTrim(chars ...string) StringTransformer {
	t.ValueTransformer = t.appendSteps(trim(trimRight, chars))
	return t
}

// ToLower converts letters to lower case.
func (t StringTransformer) ToLower() StringTransformer {
	t.ValueTransformer = t.appendSteps(lower())
	return t
}

// ToUpper converts letters to upper case.
func (t StringTransformer) ToUpper() StringTransformer {
	t.ValueTransformer = t.appendSteps(upper())
	return t
}

// Apply appends custom string transformations.
func (t StringTransformer) Apply(fns ...func(string) (string, error)) StringTransformer {
	t.ValueTransformer = t.ValueTransformer.Apply(fns...)
	return t
}

// ApplyContext appends context-aware custom string transformations.
func (t StringTransformer) ApplyContext(fns ...func(context.Context, string) (string, error)) StringTransformer {
	t.ValueTransformer = t.ValueTransformer.ApplyContext(fns...)
	return t
}
