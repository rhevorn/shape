package shape

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"time"

	"github.com/rhevorn/shape/internal/spec"
)

const defaultMaxRecursiveDepth = 64

type compiledRule func(context.Context, reflect.Value) error
type valueStepKind uint8

const (
	valueStepOption valueStepKind = iota
	valueStepTransform
	valueStepRule
)

type valueStep struct {
	kind      valueStepKind
	option    string
	fallback  reflect.Value
	transform func(context.Context, reflect.Value) (reflect.Value, error)
	check     compiledRule
}

type valuePlan struct {
	typ          reflect.Type
	element      *valuePlan
	key          *valuePlan
	fields       []compiledField
	fallback     reflect.Value
	fallbackKind string
	transforms   []func(context.Context, reflect.Value) (reflect.Value, error)
	checks       []compiledRule
	descriptors  []spec.Rule
	steps        []valueStep
	label        string
}

func copyPlan(p *valuePlan) *valuePlan { n := *p; return &n }
func zeroValue(v reflect.Value) bool {
	if v.Type() == reflect.TypeFor[time.Time]() {
		return v.Interface().(time.Time).IsZero()
	}
	return v.IsZero()
}
func finite(v reflect.Value) bool {
	if v.Kind() == reflect.Float32 || v.Kind() == reflect.Float64 {
		x := v.Float()
		return !math.IsNaN(x) && !math.IsInf(x, 0)
	}
	return true
}
func safeNumericConversion(v reflect.Value, target reflect.Type) (reflect.Value, bool) {
	if !v.IsValid() || !numericKind(v.Kind()) || !numericKind(target.Kind()) || !v.Type().ConvertibleTo(target) || !finite(v) {
		return reflect.Value{}, false
	}
	out := v.Convert(target)
	if !finite(out) {
		return reflect.Value{}, false
	}
	if target.Kind() == reflect.Float32 || target.Kind() == reflect.Float64 {
		return out, true
	}
	return out, numericRat(v).Cmp(numericRat(out)) == 0
}
func numericRat(v reflect.Value) *big.Rat {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return new(big.Rat).SetInt64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := new(big.Int).SetUint64(v.Uint())
		return new(big.Rat).SetInt(n)
	case reflect.Float32, reflect.Float64:
		return new(big.Rat).SetFloat64(v.Float())
	default:
		panic("shape: numericRat called with a non-number")
	}
}
func nilable(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Map
}
func addOptions(p *valuePlan, options ...spec.Option) *valuePlan {
	p = copyPlan(p)
	for _, o := range options {
		name := spec.OptionName(o)
		if name != "ifzero" && name != "ifnull" {
			panic("shape: unknown option " + name)
		}
		if p.fallbackKind != "" {
			panic("shape: only one IfZero/IfNull fallback is permitted")
		}
		if name == "ifzero" && nilable(p.typ) {
			panic("shape: IfZero does not support pointer, slice or map; use IfNull")
		}
		if name == "ifnull" && !nilable(p.typ) {
			panic("shape: IfNull requires pointer, slice or map")
		}
		v := reflect.ValueOf(spec.OptionValue(o))
		if !v.IsValid() {
			if !nilable(p.typ) {
				panic("shape: nil fallback for non-nilable type")
			}
			v = reflect.Zero(p.typ)
		}
		if v.Type() != p.typ {
			converted, ok := safeNumericConversion(v, p.typ)
			if !ok {
				panic(fmt.Sprintf("shape: fallback must be %v, got %v", p.typ, v.Type()))
			}
			v = converted
		}
		if !finite(v) {
			panic("shape: fallback must be finite")
		}
		snapshot, err := cloneValue(context.Background(), v, 0, true)
		if err != nil {
			panic("shape: invalid fallback: " + err.Error())
		}
		p.fallbackKind = name
		p.fallback = snapshot
		p.steps = appendCopy(p.steps, valueStep{kind: valueStepOption, option: name, fallback: snapshot})
	}
	return p
}
func addRules(p *valuePlan, rules ...spec.Rule) *valuePlan {
	p = copyPlan(p)
	for _, r := range rules {
		check := compileRule(p.typ, r)
		p.checks = appendCopy(p.checks, check)
		p.descriptors = appendCopy(p.descriptors, r)
		p.steps = appendCopy(p.steps, valueStep{kind: valueStepRule, check: check})
	}
	return p
}
func cloneValue(ctx context.Context, v reflect.Value, depth int, strict bool) (reflect.Value, error) {
	if err := ctx.Err(); err != nil {
		return reflect.Value{}, err
	}
	if depth >= defaultMaxRecursiveDepth {
		return reflect.Value{}, errors.New("shape: copy depth exceeded (cyclic or deeply nested value)")
	}
	if v.Type() == reflect.TypeFor[time.Time]() {
		return v, nil
	}
	out := reflect.New(v.Type()).Elem()
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return out, nil
		}
		c, err := cloneValue(ctx, v.Elem(), depth+1, strict)
		if err != nil {
			return out, err
		}
		out.Set(reflect.New(v.Type().Elem()))
		out.Elem().Set(c)
	case reflect.Slice:
		if v.IsNil() {
			return out, nil
		}
		out = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			c, err := cloneValue(ctx, v.Index(i), depth+1, strict)
			if err != nil {
				return out, err
			}
			out.Index(i).Set(c)
		}
	case reflect.Map:
		if v.IsNil() {
			return out, nil
		}
		out = reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k, err := cloneValue(ctx, iter.Key(), depth+1, strict)
			if err != nil {
				return out, err
			}
			c, err := cloneValue(ctx, iter.Value(), depth+1, strict)
			if err != nil {
				return out, err
			}
			out.SetMapIndex(k, c)
		}
	case reflect.Struct:
		out.Set(v)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !out.Field(i).CanSet() || v.Type().Field(i).PkgPath != "" {
				if strict && !immutableType(f.Type()) {
					return out, errors.New("unexported mutable default field")
				}
				continue
			}
			c, err := cloneValue(ctx, f, depth+1, strict)
			if err != nil {
				return out, err
			}
			out.Field(i).Set(c)
		}
	case reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		if strict {
			return out, errors.New("unsupported default type: " + v.Type().String())
		}
		out.Set(v)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			c, err := cloneValue(ctx, v.Index(i), depth+1, strict)
			if err != nil {
				return out, err
			}
			out.Index(i).Set(c)
		}
	default:
		out.Set(v)
	}
	return out, nil
}
func immutableType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !immutableType(t.Field(i).Type) {
				return false
			}
		}
		return true
	case reflect.Array:
		return immutableType(t.Elem())
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	}
	return true
}
