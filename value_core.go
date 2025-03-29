package shape

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"

	"github.com/rhevorn/shape/internal/reflectclone"
	"github.com/rhevorn/shape/internal/spec"
)

// defaultMaxRecursiveDepth bounds plan traversal. It is separate from
// reflectclone.MaxDepth, which bounds copies of a single value.
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
func zeroValue(v reflect.Value) bool   { return reflectclone.IsZero(v) }
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
		snapshot, err := cloneValue(context.Background(), v, true)
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

// cloneValue delegates to the shared implementation so the root package and
// transform cannot drift apart on what "detached copy" means.
func cloneValue(ctx context.Context, v reflect.Value, strict bool) (reflect.Value, error) {
	out, err := reflectclone.Clone(ctx, v, strict)
	switch {
	case err == nil:
		return out, nil
	case errors.Is(err, reflectclone.ErrDepthExceeded):
		return reflect.Value{}, errors.New("shape: copy depth exceeded (cyclic or deeply nested value)")
	case errors.Is(err, reflectclone.ErrUnsupportedType):
		return reflect.Value{}, fmt.Errorf("shape: unsupported default type: %v", err)
	default:
		return reflect.Value{}, err
	}
}
func immutableType(t reflect.Type) bool { return reflectclone.ImmutableType(t) }
