package shape

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
)

// PathKind identifies the kind of one validation path segment.
type PathKind uint8

const (
	// PathField identifies an object or map key.
	PathField PathKind = iota
	// PathIndex identifies a collection index.
	PathIndex
)

// PathSegment is one typed component of a validation path.
// Exactly one of Key and Index is meaningful, according to Kind.
type PathSegment struct {
	Kind  PathKind `json:"kind"`
	Key   string   `json:"key,omitempty"`
	Index int      `json:"index,omitempty"`
}

// FieldPath returns a field path segment.
func FieldPath(key string) PathSegment {
	return PathSegment{Kind: PathField, Key: key}
}

// IndexPath returns an index path segment.
func IndexPath(index int) PathSegment {
	return PathSegment{Kind: PathIndex, Index: index}
}

// Path is a structured location in an input value.
type Path []PathSegment

// String formats a path using familiar field and index notation.
// The root path is formatted as "$".
func (p Path) String() string {
	if len(p) == 0 {
		return "$"
	}

	var b strings.Builder
	for i, segment := range p {
		switch segment.Kind {
		case PathField:
			if isIdentifier(segment.Key) {
				if i > 0 {
					b.WriteByte('.')
				}
				b.WriteString(segment.Key)
				continue
			}
			b.WriteByte('[')
			b.WriteString(strconv.Quote(segment.Key))
			b.WriteByte(']')
		case PathIndex:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(segment.Index))
			b.WriteByte(']')
		default:
			b.WriteString("[?]")
		}
	}
	return b.String()
}

// MarshalJSON presents a path as a compact array of field names and indexes
// while retaining typed segments in memory.
func (p Path) MarshalJSON() ([]byte, error) {
	segments := make([]any, len(p))
	for i, segment := range p {
		if segment.Kind == PathIndex {
			segments[i] = segment.Index
		} else {
			segments[i] = segment.Key
		}
	}
	return json.Marshal(segments)
}

func (p Path) prefixed(segment PathSegment) Path {
	result := make(Path, len(p)+1)
	result[0] = segment
	copy(result[1:], p)
	return result
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}
