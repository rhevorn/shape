package transform

import "context"

type StringTransformer struct{ ValueTransformer[string] }

func (t StringTransformer) IfZero(v string) StringTransformer {
	t.ValueTransformer = t.ValueTransformer.IfZero(v)
	return t
}
func (t StringTransformer) Trim(chars ...string) StringTransformer {
	t.ValueTransformer = t.append(trim(0, chars))
	return t
}
func (t StringTransformer) LTrim(chars ...string) StringTransformer {
	t.ValueTransformer = t.append(trim(-1, chars))
	return t
}
func (t StringTransformer) RTrim(chars ...string) StringTransformer {
	t.ValueTransformer = t.append(trim(1, chars))
	return t
}
func (t StringTransformer) ToLower() StringTransformer {
	t.ValueTransformer = t.append(lower())
	return t
}
func (t StringTransformer) ToUpper() StringTransformer {
	t.ValueTransformer = t.append(upper())
	return t
}
func (t StringTransformer) Apply(fns ...func(string) (string, error)) StringTransformer {
	t.ValueTransformer = t.ValueTransformer.Apply(fns...)
	return t
}
func (t StringTransformer) ApplyContext(fns ...func(context.Context, string) (string, error)) StringTransformer {
	t.ValueTransformer = t.ValueTransformer.ApplyContext(fns...)
	return t
}
