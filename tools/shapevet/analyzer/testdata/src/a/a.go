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

func build() {
	_ = shape.Struct[Good]()
	_ = shape.Struct[Unsupported]()    // want "unsupported field type"
	_ = shape.Struct[Duplicate]()      // want "duplicate field name value"
	_ = shape.Struct[PointerToSlice]() // want "unsupported pointer field type"
	_ = shape.Struct[Embedded]()       // want "anonymous fields are unsupported"
	_ = shape.Struct[Recursive]()      // want "recursive type is unsupported"
	var bind Unsupported
	_ = shape.BindJSON(&bind, nil) // want "unsupported field type"

	_ = shape.New[Explicit](
		shape.String("Name").Trim().NotEmpty(),
		shape.Int("Age").Min(18),
	)
	_ = shape.New[Explicit](shape.String("Missing")) // want "target has no direct field Missing"
	_ = shape.New[Explicit](shape.Int("Name"))       // want "field Name has type string, contract has type int"
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
