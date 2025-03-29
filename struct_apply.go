package shape

import (
	"context"
	"fmt"
	"github.com/rhevorn/shape/internal/spec"
	"github.com/rhevorn/shape/types"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
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
func applyTag(p *valuePlan, text string) (out *valuePlan, err error) {
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
	p = copyPlan(p)
	labelSeen := false
	for _, o := range opts {
		target := p
		outer := o.name == "ifzero" || o.name == "ifnull" || o.name == "notnull" || o.name == "notempty" || o.name == "label"
		pointerInner := p.typ.Kind() == reflect.Pointer && !outer
		if pointerInner {
			if p.element == nil {
				return nil, fmt.Errorf("missing pointer element")
			}
			target = copyPlan(p.element)
		}
		isFlag := map[string]bool{"notnull": true, "notempty": true, "positive": true, "negative": true, "nonnegative": true, "unique": true, "tolower": true, "toupper": true, "email": true, "url": true, "uuid": true, "ip": true}
		if isFlag[o.name] && o.has {
			return nil, fmt.Errorf("%s takes no argument", o.name)
		}
		switch o.name {
		case "label":
			if !o.has || labelSeen {
				return nil, fmt.Errorf("invalid or duplicate label")
			}
			p.label = o.value
			labelSeen = true
		case "ifzero", "ifnull":
			if !o.has {
				return nil, fmt.Errorf("%s requires a constant", o.name)
			}
			t := p.typ
			if t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
			value, e := parseConstant(t, o.value)
			if e != nil {
				return nil, e
			}
			if p.typ.Kind() == reflect.Pointer {
				ptr := reflect.New(t)
				ptr.Elem().Set(value)
				value = ptr
			}
			p = addOptions(p, spec.NewOption(o.name, value.Interface()))
		case "trim", "ltrim", "rtrim", "tolower", "toupper":
			if target.typ.Kind() != reflect.String {
				return nil, fmt.Errorf("%s requires string", o.name)
			}
			var args []string
			if o.has {
				args = []string{o.value}
			}
			fn := func(ctx context.Context, v reflect.Value) (reflect.Value, error) {
				if err := ctx.Err(); err != nil {
					return reflect.Value{}, err
				}
				s := applyStringTransform(o.name, v.String(), args)
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
			if o.has {
				switch o.name {
				case "minlength", "maxlength", "len":
					n, e := tagCount(o.value)
					if e != nil {
						return nil, e
					}
					args = []any{n}
				case "pattern", "startswith", "endswith", "contains":
					args = []any{o.value}
				case "min", "max", "gt", "gte", "lt", "lte", "between", "oneof":
					if target.typ.Kind() == reflect.Slice || target.typ.Kind() == reflect.Map {
						n, e := tagCount(o.value)
						if e != nil {
							return nil, e
						}
						args = []any{n}
					} else {
						parts := []string{o.value}
						if o.name == "between" || o.name == "oneof" {
							parts = strings.Split(o.value, "|")
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
					return nil, fmt.Errorf("unknown or argument-less tag %s", o.name)
				}
			} else if !isFlag[o.name] {
				return nil, fmt.Errorf("unknown tag or missing argument: %s", o.name)
			}
			descriptor := spec.NewRule(o.name, args...)
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
		inner := copyPlan(p.element)
		inner.label = p.label
		p.element = inner
	}
	return p, nil
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
	return value
}

func appendCompiledTransform(p *valuePlan, fn func(context.Context, reflect.Value) (reflect.Value, error)) *valuePlan {
	p = copyPlan(p)
	p.transforms = appendCopy(p.transforms, fn)
	p.steps = appendCopy(p.steps, valueStep{kind: valueStepTransform, transform: fn})
	return p
}

func appendPointerTagTransform(p *valuePlan, inner func(context.Context, reflect.Value) (reflect.Value, error)) *valuePlan {
	p = copyPlan(p)
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
	p.transforms = appendCopy(p.transforms, outer)
	p.steps = appendCopy(p.steps, valueStep{kind: valueStepTransform, transform: outer})
	return p
}

func appendPointerTagRule(p *valuePlan, descriptor spec.Rule) *valuePlan {
	p = copyPlan(p)
	inner := copyPlan(p.element)
	check := compileRule(inner.typ, descriptor)
	inner.checks = appendCopy(inner.checks, check)
	inner.descriptors = appendCopy(inner.descriptors, descriptor)
	inner.steps = appendCopy(inner.steps, valueStep{kind: valueStepRule, check: check})
	p.element = inner
	return p
}
