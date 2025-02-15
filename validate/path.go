package validate

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
)

type PathKind uint8

const (
	PathField PathKind = iota
	PathIndex
)

type PathSegment struct {
	Kind  PathKind `json:"kind"`
	Key   string   `json:"key,omitempty"`
	Index int      `json:"index,omitempty"`
}

func FieldPath(key string) PathSegment { return PathSegment{Kind: PathField, Key: key} }
func IndexPath(index int) PathSegment  { return PathSegment{Kind: PathIndex, Index: index} }

type Path []PathSegment

func (p Path) String() string {
	if len(p) == 0 {
		return "$"
	}
	var b strings.Builder
	for i, s := range p {
		if s.Kind == PathIndex {
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(s.Index))
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

func (p Path) MarshalJSON() ([]byte, error) {
	values := make([]any, len(p))
	for i, s := range p {
		if s.Kind == PathIndex {
			values[i] = s.Index
		} else {
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
