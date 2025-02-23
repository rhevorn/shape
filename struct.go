package shape

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

type structCacheEntry struct {
	plan *valuePlan
	err  error
}

var structCache sync.Map

func taggedSchema[T any]() TaggedSpec[T] {
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Struct || t == reflect.TypeFor[time.Time]() {
		panic("shape: Struct requires an ordinary value struct")
	}
	if cached, ok := structCache.Load(t); ok {
		e := cached.(structCacheEntry)
		if e.err != nil {
			panic(e.err)
		}
		return newTaggedSpec[T](e.plan)
	}
	p, err := compileType(t, map[reflect.Type]bool{})
	actual, _ := structCache.LoadOrStore(t, structCacheEntry{p, err})
	e := actual.(structCacheEntry)
	if e.err != nil {
		panic(e.err)
	}
	return newTaggedSpec[T](e.plan)
}

type compiledField struct {
	index int
	name  string
	plan  *valuePlan
}

func compileType(t reflect.Type, active map[reflect.Type]bool) (*valuePlan, error) {
	p := &valuePlan{typ: t}
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
		fields := make([]compiledField, 0, t.NumField())
		names := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("shape")
			jsonName := strings.Split(f.Tag.Get("json"), ",")[0]
			if f.PkgPath != "" || jsonName == "-" {
				if tag != "" {
					return nil, fmt.Errorf("shape: %s has a tag but is not processed", f.Name)
				}
				continue
			}
			if f.Anonymous {
				return nil, fmt.Errorf("shape: anonymous field %s is unsupported", f.Name)
			}
			if jsonName == "" {
				jsonName = f.Name
			}
			if names[jsonName] {
				return nil, fmt.Errorf("shape: duplicate field name %s", jsonName)
			}
			names[jsonName] = true
			inner, e := compileType(f.Type, active)
			if e != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, e)
			}
			inner, e = applyTag(inner, tag)
			if e != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, e)
			}
			fields = append(fields, compiledField{i, jsonName, inner})
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
		p.key = &valuePlan{typ: t.Key()}
	default:
		return nil, fmt.Errorf("shape: unsupported field type %v", t)
	}
	return p, nil
}
