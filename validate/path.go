package validate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// PathKind identifies the kind of one validation path segment.
type PathKind uint8

const (
	// PathField identifies a struct or object field.
	PathField PathKind = iota
	// PathIndex identifies a slice index.
	PathIndex
	// PathMapKey identifies a typed map key.
	PathMapKey
)

// PathSegment is one field, slice index, or typed map key in a Path.
type PathSegment struct {
	Kind   PathKind `json:"kind"`
	Key    string   `json:"key,omitempty"`
	Index  int      `json:"index,omitempty"`
	MapKey any      `json:"mapKey,omitempty"`
}

// FieldPath creates a field segment.
func FieldPath(key string) PathSegment { return PathSegment{Kind: PathField, Key: key} }

// IndexPath creates a slice-index segment.
func IndexPath(index int) PathSegment { return PathSegment{Kind: PathIndex, Index: index} }

// MapKeyPath creates a segment that preserves the concrete map-key value.
func MapKeyPath(key any) PathSegment { return PathSegment{Kind: PathMapKey, MapKey: key} }

// Path locates a validation issue inside nested values.
type Path []PathSegment

// String renders a Go-like field and collection path.
func (p Path) String() string {
	if len(p) == 0 {
		return "$"
	}
	var b strings.Builder
	for i, s := range p {
		switch s.Kind {
		case PathIndex:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(s.Index))
			b.WriteByte(']')
			continue
		case PathMapKey:
			b.WriteByte('[')
			if key, ok := s.MapKey.(string); ok {
				b.WriteString(strconv.Quote(key))
			} else {
				b.WriteString(fmt.Sprint(s.MapKey))
			}
			b.WriteByte(']')
			continue
		}
		if identifier(s.Key) {
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(s.Key)
		} else {
			b.WriteByte('[')
			b.WriteString(strconv.Quote(s.Key))
			b.WriteByte(']')
		}
	}
	return b.String()
}

// MarshalJSON encodes fields and map keys as their values and indexes as numbers.
func (p Path) MarshalJSON() ([]byte, error) {
	values := make([]any, len(p))
	for i, s := range p {
		switch s.Kind {
		case PathIndex:
			values[i] = s.Index
		case PathMapKey:
			values[i] = s.MapKey
		default:
			values[i] = s.Key
		}
	}
	return json.Marshal(values)
}

func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}
