// Package valuesdecode decodes form and query parameters into structs.
package valuesdecode

import (
	"context"
	"encoding"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rhevorn/shape/internal/fieldmeta"
	shapetypes "github.com/rhevorn/shape/types"
	"github.com/rhevorn/shape/validate"
)

type Error struct {
	Path validate.Path
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

type scalarDecoder func(reflect.Value, string) error

type field struct {
	key    string
	path   validate.Path
	index  []int
	slice  bool
	decode scalarDecoder
}

type plan struct {
	fields []field
	keys   map[string]bool
	groups map[string]validate.Path
	owned  bool
}

type cacheKey struct {
	typ    reflect.Type
	source fieldmeta.Source
}

type cacheEntry struct {
	plan *plan
	err  error
}

var cache sync.Map
var textType = reflect.TypeFor[encoding.TextUnmarshaler]()

// Decode returns private storage unless a custom text decoder can introduce aliases.
// Unsupported type definitions panic; invalid parameter values return errors.
func Decode[T any](ctx context.Context, values url.Values, source fieldmeta.Source, strict bool) (T, bool, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, false, err
	}
	p := cachedPlan(reflect.TypeFor[T](), source)
	out := reflect.New(reflect.TypeFor[T]()).Elem()
	for _, f := range p.fields {
		if err := ctx.Err(); err != nil {
			return zero, false, err
		}
		items, present := values[f.key]
		if !present {
			continue
		}
		if len(items) == 0 {
			return zero, false, &Error{Path: f.path, Err: errors.New("parameter has no values")}
		}
		if !f.slice && len(items) != 1 {
			return zero, false, &Error{Path: f.path, Err: errors.New("multiple values for a scalar parameter")}
		}
		dst := fieldAt(out, f.index)
		if f.slice {
			dst.Set(reflect.MakeSlice(dst.Type(), len(items), len(items)))
			for i, text := range items {
				if err := ctx.Err(); err != nil {
					return zero, false, err
				}
				if err := f.decode(dst.Index(i), text); err != nil {
					path := append(append(validate.Path(nil), f.path...), validate.IndexPath(i))
					return zero, false, &Error{Path: path, Err: err}
				}
			}
		} else if err := f.decode(dst, items[0]); err != nil {
			return zero, false, &Error{Path: f.path, Err: err}
		}
	}
	// Map iteration order must not decide which unknown or malformed key is reported.
	var invalid string
	found := false
	for key := range values {
		if err := ctx.Err(); err != nil {
			return zero, false, err
		}
		_, group := p.groups[key]
		if (group || strict && !p.keys[key]) && (!found || key < invalid) {
			invalid, found = key, true
		}
	}
	if found {
		if path, group := p.groups[invalid]; group {
			return zero, false, &Error{Path: path, Err: errors.New("expected dotted fields for a nested struct")}
		}
		return zero, false, &Error{Path: validate.Path{validate.FieldPath(invalid)}, Err: errors.New("unknown parameter")}
	}
	if err := ctx.Err(); err != nil {
		return zero, false, err
	}
	return out.Interface().(T), p.owned, nil
}

// Check validates and caches the decoding configuration before reading input.
func Check(t reflect.Type, source fieldmeta.Source) { _ = cachedPlan(t, source) }

// Presence identifies submitted top-level Go fields, including empty or zero values.
// Call after Decode succeeds so malformed groups and duplicate scalars cannot pass.
func Presence(t reflect.Type, values url.Values, source fieldmeta.Source) []bool {
	p := cachedPlan(t, source)
	present := make([]bool, t.NumField())
	for _, f := range p.fields {
		if _, ok := values[f.key]; ok {
			present[f.index[0]] = true
		}
	}
	return present
}

func cachedPlan(t reflect.Type, source fieldmeta.Source) *plan {
	key := cacheKey{t, source}
	if item, ok := cache.Load(key); ok {
		e := item.(cacheEntry)
		if e.err != nil {
			panic(e.err)
		}
		return e.plan
	}
	p := &plan{keys: make(map[string]bool), groups: make(map[string]validate.Path), owned: true}
	var err error
	if t.Kind() != reflect.Struct || t == reflect.TypeFor[time.Time]() {
		err = errors.New("target must be an ordinary value struct")
	} else {
		err = p.compile(t, source, nil, nil, "", make(map[reflect.Type]bool))
	}
	if err != nil {
		err = fmt.Errorf("shape: invalid %s schema: %w", source, err)
	}
	actual, _ := cache.LoadOrStore(key, cacheEntry{p, err})
	e := actual.(cacheEntry)
	if e.err != nil {
		panic(e.err)
	}
	return e.plan
}

func (p *plan) compile(t reflect.Type, source fieldmeta.Source, indexes []int, path validate.Path, prefix string, active map[reflect.Type]bool) error {
	if len(indexes) >= 64 || active[t] {
		return fmt.Errorf("recursive or excessively deep struct %v", t)
	}
	active[t] = true
	defer delete(active, t)
	names := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := fieldmeta.Resolve(f.Name, f.Tag).For(source)
		if name == "" {
			continue
		}
		if f.Anonymous {
			return fmt.Errorf("anonymous field %s is unsupported", f.Name)
		}
		if !fieldmeta.ValidInputName(name) {
			return fmt.Errorf("invalid field name %q", name)
		}
		if names[name] {
			return fmt.Errorf("duplicate field name %q", prefix+name)
		}
		names[name] = true
		key := prefix + name
		index := append(append([]int(nil), indexes...), i)
		fieldPath := append(append(validate.Path(nil), path...), validate.FieldPath(name))
		typ := f.Type
		base := typ
		if base.Kind() == reflect.Pointer {
			base = base.Elem()
		}
		if base.Kind() == reflect.Struct && !reflect.PointerTo(base).Implements(textType) {
			p.groups[key] = fieldPath
			if err := p.compile(base, source, index, fieldPath, key+".", active); err != nil {
				return err
			}
			continue
		}
		many := typ.Kind() == reflect.Slice && !reflect.PointerTo(typ).Implements(textType)
		if many {
			typ = typ.Elem()
		}
		decode, owned, err := compileScalar(typ)
		if err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}
		p.owned = p.owned && owned
		p.keys[key] = true
		p.fields = append(p.fields, field{key: key, path: fieldPath, index: index, slice: many, decode: decode})
	}
	return nil
}

func fieldAt(root reflect.Value, index []int) reflect.Value {
	v := root
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()).Convert(v.Type()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

func compileScalar(t reflect.Type) (scalarDecoder, bool, error) {
	// Keep duration text support local: TextUnmarshaler would also change JSON map-key decoding.
	if t == reflect.TypeFor[shapetypes.Duration]() {
		return func(dst reflect.Value, text string) error {
			duration, err := time.ParseDuration(text)
			if err == nil {
				dst.SetInt(int64(duration))
			}
			return err
		}, true, nil
	}
	if t.Kind() == reflect.Pointer {
		if t.Elem().Kind() == reflect.Pointer || t.Elem().Kind() == reflect.Slice || t.Elem().Kind() == reflect.Map {
			return nil, false, fmt.Errorf("unsupported pointer type %v", t)
		}
		decode, owned, err := compileScalar(t.Elem())
		if err != nil {
			return nil, false, err
		}
		return func(dst reflect.Value, text string) error {
			v := reflect.New(t.Elem())
			if err := decode(v.Elem(), text); err != nil {
				return err
			}
			dst.Set(v.Convert(t))
			return nil
		}, owned, nil
	}
	if reflect.PointerTo(t).Implements(textType) {
		return func(dst reflect.Value, text string) error {
			return dst.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(text))
		}, false, nil
	}
	switch t.Kind() {
	case reflect.String:
		return func(dst reflect.Value, text string) error { dst.SetString(text); return nil }, true, nil
	case reflect.Bool:
		return func(dst reflect.Value, text string) error {
			switch text {
			case "true", "1":
				dst.SetBool(true)
			case "false", "0":
				dst.SetBool(false)
			default:
				return errors.New("expected true, false, 1, or 0")
			}
			return nil
		}, true, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bits := t.Bits()
		return func(dst reflect.Value, text string) error {
			n, err := strconv.ParseInt(text, 10, bits)
			if err == nil {
				dst.SetInt(n)
			}
			return err
		}, true, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bits := t.Bits()
		return func(dst reflect.Value, text string) error {
			n, err := strconv.ParseUint(text, 10, bits)
			if err == nil {
				dst.SetUint(n)
			}
			return err
		}, true, nil
	case reflect.Float32, reflect.Float64:
		bits := t.Bits()
		return func(dst reflect.Value, text string) error {
			// ParseFloat also accepts hexadecimal floats and underscores; parameters use decimal syntax.
			if strings.ContainsAny(text, "xXpP_") {
				return errors.New("expected a decimal number")
			}
			n, err := strconv.ParseFloat(text, bits)
			if err != nil {
				return err
			}
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return errors.New("expected a finite number")
			}
			dst.SetFloat(n)
			return nil
		}, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported parameter type %v; use a scalar, scalar slice, nested struct, or encoding.TextUnmarshaler", t)
	}
}
