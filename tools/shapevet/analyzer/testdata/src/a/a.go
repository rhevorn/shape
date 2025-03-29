package a

import (
	"github.com/rhevorn/shape"
	shapetypes "github.com/rhevorn/shape/types"
	"time"
)

type Good struct {
	Name    string              `shape:"ifzero=guest,trim,notempty,maxlength=50"`
	Mode    string              `shape:"oneof=read|write"`
	Items   []string            `shape:"len=2"`
	Timeout shapetypes.Duration `shape:"ifzero=30s,min=1s"`
	Enabled *bool               `shape:"ifnull=true"`
}

type Bad struct {
	Unknown string              `shape:"unknown"`      // want "unknown option unknown"
	Null    int                 `shape:"ifnull=1"`     // want "ifnull requires"
	Trim    int                 `shape:"trim"`         // want "trim requires string"
	Regexp  string              `shape:"pattern='['"`  // want "invalid pattern"
	Time    shapetypes.Duration `shape:"min=tomorrow"` // want "invalid duration"
	Comma   string              `shape:"trim,"`        // want "trailing comma"
	Bool    bool                `shape:"ifzero=yes"`   // want "boolean fallback"
	Tiny    int8                `shape:"max=999"`      // want "invalid or overflowing number"
	Items   []string            `shape:"min=-1"`       // want "invalid non-negative collection length"
	Bounds  int                 `shape:"between=2|1"`  // want "minimum exceeds maximum"
	Float   float64             `shape:"ifzero=NaN"`   // want "invalid or overflowing number"
	Empty   int                 `shape:"notempty"`     // want "notempty requires string, pointer, slice, or map"
	ZeroPtr *int                `shape:"ifzero=1"`     // want "ifzero literal is unsupported"
}

type Unsupported struct{ Dynamic any }

type Duplicate struct {
	First  string `json:"value"`
	Second string `json:"value"`
}

type PointerToSlice struct{ Values *[]string }

type Embedded struct{ Good }

type Recursive struct{ Next *Recursive }

type Explicit struct {
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Hidden string `json:"-"`
}

// Linter/runtime agreement fixtures. Every declaration below compiles and
// runs, so the analyzer must stay silent; a stray diagnostic with no want
// comment fails the test.
type NamedSlice []string

type NamedMap map[string]int

type Graph struct {
	Tags   NamedSlice `json:"tags"`
	Scores NamedMap   `json:"scores"`
}

// An empty oneof candidate is a legitimate value for a string.
type EmptyCandidate struct {
	OnlyEmpty string `shape:"oneof=''"`
	Hole      string `shape:"oneof=a||b"`
}

// A float map key compiles but panics at construction, so it must be reported
// exactly where the runtime rejects it.
type FloatKey struct {
	Ratings map[float64]int `json:"ratings"`
}

// A tag on a type-parameter field can only be judged once the type is
// instantiated, so the declaration scan must leave it alone.
type Box[T any] struct {
	Value T `json:"value" shape:"min=1"`
}

func build() {
	_ = shape.Struct[Good]()
	_ = shape.Struct[Unsupported]()    // want "unsupported field type"
	_ = shape.Struct[Duplicate]()      // want "duplicate field name value"
	_ = shape.Struct[PointerToSlice]() // want "unsupported pointer field type"
	_ = shape.Struct[Embedded]()       // want "anonymous fields are unsupported"
	_ = shape.Struct[Recursive]()      // want "recursive type is unsupported"
	_ = shape.Struct[FloatKey]()       // want "map key must be string or integer"
	_ = shape.Struct[Graph]()
	_ = shape.Struct[EmptyCandidate]()
	var bind Unsupported
	_ = shape.BindJSON(&bind, nil) // want "unsupported field type"

	_ = shape.New[Explicit](
		shape.String("Name").Trim().NotEmpty(),
		shape.Int("Age").Min(18),
	)
	_ = shape.New[Explicit](shape.String("Missing")) // want "target has no direct field Missing"
	_ = shape.New[Explicit](shape.Int("Name"))       // want "field Name has type string, schema has type int"
	_ = shape.New[Explicit](
		shape.String("Name"),
		shape.String("Name"), // want "duplicate field Name"
	)
	_ = shape.New[Explicit](shape.String("Hidden"))        // want "field Hidden is excluded from JSON"
	_ = shape.New[Explicit](shape.String())                // want "field name must not be empty"
	_ = shape.New[Explicit](shape.String("Name", "Extra")) // want "field factory accepts at most one name"
	_ = shape.New[time.Time]()                             // want "target must be an ordinary value struct"
	_ = shape.Struct[time.Time]()                          // want "target must be an ordinary value struct"
}

func genericBind[T any](target *T, data []byte) error {
	return shape.BindJSON(target, data)
}
