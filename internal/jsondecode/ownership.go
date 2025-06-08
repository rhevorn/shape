package jsondecode

import (
	"encoding"
	"encoding/json"
	"reflect"
	"sync"
)

var ownershipCache sync.Map
var jsonUnmarshaler = reflect.TypeFor[json.Unmarshaler]()
var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

// OwnsStorage reports whether decoding a zero value of t allocates all mutable
// storage. Custom decoders and interface values may refer to external storage.
func OwnsStorage(t reflect.Type) bool {
	if cached, ok := ownershipCache.Load(t); ok {
		return cached.(bool)
	}
	owned := ownsStorage(t, make(map[reflect.Type]bool))
	ownershipCache.Store(t, owned)
	return owned
}

func ownsStorage(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	codec := func(t reflect.Type) bool { return t.Implements(jsonUnmarshaler) || t.Implements(textUnmarshaler) }
	if codec(t) || t.Kind() != reflect.Pointer && codec(reflect.PointerTo(t)) {
		return false
	}
	switch t.Kind() {
	case reflect.Interface:
		return false
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return ownsStorage(t.Elem(), seen)
	case reflect.Map:
		return ownsStorage(t.Key(), seen) && ownsStorage(t.Elem(), seen)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !ownsStorage(t.Field(i).Type, seen) {
				return false
			}
		}
	}
	return true
}
