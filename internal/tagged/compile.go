package tagged

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/rhevorn/shape/internal/fieldmeta"
)

type structCacheEntry struct {
	plan *Plan
	err  error
}

var structCache sync.Map

// Compile derives and caches an immutable tag plan for t.
func Compile(t reflect.Type) (*Plan, error) {
	if t.Kind() != reflect.Struct || t == reflect.TypeFor[time.Time]() {
		return nil, fmt.Errorf("shape: FromTags requires an ordinary value struct")
	}
	if cached, ok := structCache.Load(t); ok {
		e := cached.(structCacheEntry)
		return e.plan, e.err
	}
	p, err := compileType(t, map[reflect.Type]bool{})
	if err == nil {
		prepare(p)
	}
	actual, _ := structCache.LoadOrStore(t, structCacheEntry{p, err})
	e := actual.(structCacheEntry)
	return e.plan, e.err
}

type tagField struct {
	index  int
	names  fieldmeta.Names
	quoted bool
	plan   *Plan
}

func compileType(t reflect.Type, active map[reflect.Type]bool) (*Plan, error) {
	p := &Plan{typ: t}
	if t == reflect.TypeFor[time.Time]() || t == durationType {
		return p, nil
	}
	if active[t] {
		return nil, fmt.Errorf("shape: recursive tag type %v is unsupported", t)
	}
	active[t] = true
	defer delete(active, t)
	switch t.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return p, nil
	case reflect.Pointer:
		if t.Elem().Kind() == reflect.Pointer || t.Elem().Kind() == reflect.Slice || t.Elem().Kind() == reflect.Map {
			return nil, fmt.Errorf("shape: unsupported pointer tag type %v", t)
		}
		inner, e := compileType(t.Elem(), active)
		if e != nil {
			return nil, e
		}
		p.element = inner
	case reflect.Struct:
		fields := make([]tagField, 0, t.NumField())
		names := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("shape")
			fieldNames := fieldmeta.Resolve(f.Name, f.Tag)
			if f.PkgPath != "" {
				if tag != "" {
					return nil, fmt.Errorf("shape: %s has a tag but is not processed", f.Name)
				}
				continue
			}
			if f.Anonymous {
				return nil, fmt.Errorf("shape: anonymous field %s is unsupported", f.Name)
			}
			if names[fieldNames.Default] {
				return nil, fmt.Errorf("shape: duplicate field name %s", fieldNames.Default)
			}
			names[fieldNames.Default] = true
			inner, e := compileType(f.Type, active)
			if e != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, e)
			}
			inner, e = applyTag(inner, tag)
			if e != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, e)
			}
			fields = append(fields, tagField{index: i, names: fieldNames, plan: inner, quoted: jsonQuoted(f)})
		}
		p.fields = fields
	case reflect.Slice:
		inner, e := compileType(t.Elem(), active)
		if e != nil {
			return nil, e
		}
		p.element = inner
	case reflect.Map:
		if t.Key().Kind() != reflect.String && !(t.Key().Kind() >= reflect.Int && t.Key().Kind() <= reflect.Uint64 && t.Key().Kind() != reflect.Uintptr) {
			return nil, fmt.Errorf("shape: unsupported map key %v", t.Key())
		}
		inner, e := compileType(t.Elem(), active)
		if e != nil {
			return nil, e
		}
		p.element = inner
	default:
		return nil, fmt.Errorf("shape: unsupported field type %v", t)
	}
	return p, nil
}

func jsonQuoted(field reflect.StructField) bool {
	for _, option := range strings.Split(field.Tag.Get("json"), ",")[1:] {
		if option == "string" {
			return true
		}
	}
	return false
}
