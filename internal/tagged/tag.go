package tagged

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/rhevorn/shape/internal/spec"
	"github.com/rhevorn/shape/types"
)

var durationType = reflect.TypeFor[types.Duration]()

func parseConstant(t reflect.Type, text string) (reflect.Value, error) {
	v := reflect.New(t).Elem()
	if t == durationType {
		n, e := time.ParseDuration(text)
		v.SetInt(int64(n))
		return v, e
	}
	if t == reflect.TypeFor[time.Time]() {
		n, e := time.Parse(time.RFC3339Nano, text)
		return reflect.ValueOf(n), e
	}
	switch t.Kind() {
	case reflect.String:
		v.SetString(text)
	case reflect.Bool:
		if text != "true" && text != "false" {
			return v, fmt.Errorf("expected true or false")
		}
		v.SetBool(text == "true")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, e := strconv.ParseInt(text, 10, t.Bits())
		if e != nil {
			return v, e
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, e := strconv.ParseUint(text, 10, t.Bits())
		if e != nil {
			return v, e
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		if strings.ContainsAny(text, "xXpP_") {
			return v, fmt.Errorf("expected decimal float")
		}
		n, e := strconv.ParseFloat(text, t.Bits())
		if e != nil {
			return v, e
		}
		if math.IsInf(n, 0) || math.IsNaN(n) {
			return v, fmt.Errorf("non-finite constant")
		}
		v.SetFloat(n)
	default:
		return v, fmt.Errorf("unsupported constant type %v", t)
	}
	return v, nil
}
func applyTag(p *Plan, text string) (out *Plan, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("shape: %v", r)
		}
	}()
	opts, e := parseShapeTag(text)
	if e != nil {
		return nil, e
	}
	p = copyTagPlan(p)
	labelSeen := false
	for _, o := range opts {
		target := p
		outer := o.Name == "ifzero" || o.Name == "ifnull" || o.Name == "notnull" || o.Name == "notempty" || o.Name == "label"
		pointerInner := p.typ.Kind() == reflect.Pointer && !outer
		if pointerInner {
			if p.element == nil {
				return nil, fmt.Errorf("missing pointer element")
			}
			target = copyTagPlan(p.element)
		}
		if flagTag(o.Name) && o.HasValue {
			return nil, fmt.Errorf("%s takes no argument", o.Name)
		}
		switch o.Name {
		case "label":
			if !o.HasValue || labelSeen {
				return nil, fmt.Errorf("invalid or duplicate label")
			}
			p.label = o.Value
			labelSeen = true
		case "ifzero", "ifnull":
			if !o.HasValue {
				return nil, fmt.Errorf("%s requires a constant", o.Name)
			}
			t := p.typ
			if t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
			value, e := parseConstant(t, o.Value)
			if e != nil {
				return nil, e
			}
			if p.typ.Kind() == reflect.Pointer {
				ptr := reflect.New(t)
				ptr.Elem().Set(value)
				value = ptr
			}
			p = addOptions(p, spec.NewOption(o.Name, value.Interface()))
		case "trim", "ltrim", "rtrim", "tolower", "toupper":
			if target.typ.Kind() != reflect.String {
				return nil, fmt.Errorf("%s requires string", o.Name)
			}
			var args []string
			if o.HasValue {
				args = []string{o.Value}
			}
			name := o.Name
			fn := func(ctx context.Context, v reflect.Value) (reflect.Value, error) {
				if err := ctx.Err(); err != nil {
					return reflect.Value{}, err
				}
				s := applyStringTransform(name, v.String(), args)
				out := reflect.New(v.Type()).Elem()
				out.SetString(s)
				return out, nil
			}
			if pointerInner {
				p = appendPointerTagTransform(p, fn)
			} else {
				p = appendCompiledTransform(p, fn)
			}
		default:
			var args []any
			if o.HasValue {
				switch o.Name {
				case "minlength", "maxlength", "len":
					n, e := tagCount(o.Value)
					if e != nil {
						return nil, e
					}
					args = []any{n}
				case "pattern", "startswith", "endswith", "contains":
					args = []any{o.Value}
				case "min", "max", "gt", "gte", "lt", "lte", "between", "oneof":
					if target.typ.Kind() == reflect.Slice || target.typ.Kind() == reflect.Map {
						n, e := tagCount(o.Value)
						if e != nil {
							return nil, e
						}
						args = []any{n}
					} else {
						parts := []string{o.Value}
						if o.Name == "between" || o.Name == "oneof" {
							parts = strings.Split(o.Value, "|")
						}
						for _, part := range parts {
							v, e := parseConstant(target.typ, strings.TrimSpace(part))
							if e != nil {
								return nil, e
							}
							args = append(args, v.Interface())
						}
					}
				default:
					return nil, fmt.Errorf("unknown or argument-less tag %s", o.Name)
				}
			} else if !flagTag(o.Name) {
				return nil, fmt.Errorf("unknown tag or missing argument: %s", o.Name)
			}
			descriptor := spec.NewRule(o.Name, args...)
			if pointerInner {
				p = appendPointerTagRule(p, descriptor)
			} else {
				p = addRules(p, descriptor)
			}
		}
	}
	// label is an outer tag, so it lands on the pointer plan, while rules for a
	// pointer field are attached to a copy of the element plan and rendering
	// reads the label of the plan being walked. Without this, `label=X,min=1`
	// silently loses its label on *T fields but not on T.
	if p.typ.Kind() == reflect.Pointer && p.label != "" && p.element != nil && p.element.label == "" {
		inner := copyTagPlan(p.element)
		inner.label = p.label
		p.element = inner
	}
	return p, nil
}

func flagTag(name string) bool {
	switch name {
	case "notnull", "notempty", "positive", "negative", "nonnegative", "unique", "tolower", "toupper", "email", "url", "uuid", "ip":
		return true
	default:
		return false
	}
}

func applyStringTransform(name, value string, args []string) string {
	if len(args) == 0 {
		switch name {
		case "trim":
			return strings.TrimSpace(value)
		case "ltrim":
			return strings.TrimLeftFunc(value, unicode.IsSpace)
		case "rtrim":
			return strings.TrimRightFunc(value, unicode.IsSpace)
		case "tolower":
			return strings.ToLower(value)
		case "toupper":
			return strings.ToUpper(value)
		}
	}
	switch name {
	case "trim":
		return strings.Trim(value, args[0])
	case "ltrim":
		return strings.TrimLeft(value, args[0])
	case "rtrim":
		return strings.TrimRight(value, args[0])
	}
	panic("shape: uncompiled string transform " + name)
}

func appendCompiledTransform(p *Plan, fn func(context.Context, reflect.Value) (reflect.Value, error)) *Plan {
	p = copyTagPlan(p)
	p.hasTransform = true
	p.steps = appendCopy(p.steps, tagStep{kind: tagStepTransform, transform: fn})
	return p
}

func appendPointerTagTransform(p *Plan, inner func(context.Context, reflect.Value) (reflect.Value, error)) *Plan {
	p = copyTagPlan(p)
	outer := func(ctx context.Context, value reflect.Value) (reflect.Value, error) {
		if value.IsNil() {
			return value, nil
		}
		out, err := inner(ctx, value.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		result := reflect.New(value.Type().Elem())
		result.Elem().Set(out)
		return result, nil
	}
	p.hasTransform = true
	p.steps = appendCopy(p.steps, tagStep{kind: tagStepTransform, transform: outer})
	return p
}

func appendPointerTagRule(p *Plan, descriptor spec.Rule) *Plan {
	p = copyTagPlan(p)
	inner := copyTagPlan(p.element)
	check := compileRule(inner.typ, descriptor)
	inner.descriptors = appendCopy(inner.descriptors, descriptor)
	inner.steps = appendCopy(inner.steps, tagStep{kind: tagStepRule, check: check})
	p.element = inner
	return p
}
