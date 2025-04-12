package negative

import (
	"context"
	"strings"
	"time"

	"github.com/rhevorn/shape"
	shapetypes "github.com/rhevorn/shape/types"
)

type InvalidSyntax struct {
	Leading      string `shape:",trim"`          // want "empty tag name"
	Name         string `shape:"min-length=1"`   // want "invalid tag name"
	MissingValue string `shape:"minlength="`     // want "empty or invalid unquoted tag value"
	Unterminated string `shape:"trim='x"`        // want "unterminated tag quote"
	Escape       string `shape:"trim='\\q'"`     // want "invalid escape"
	AfterQuote   string `shape:"trim='x'oops"`   // want "expected comma"
	Trailing     string `shape:"trim,"`          // want "trailing comma"
	EmptyItem    string `shape:"trim,,notempty"` // want "empty tag name"
}

type InvalidRules struct {
	Unknown           string            `shape:"unknown"`           // want "unknown option unknown"
	LabelMissing      string            `shape:"label"`             // want "label requires one value"
	LabelDuplicate    string            `shape:"label=a,label=b"`   // want "label requires one value and may appear once"
	FallbackMissing   string            `shape:"ifzero"`            // want "invalid or duplicate fallback"
	FallbackDuplicate string            `shape:"ifzero=a,ifzero=b"` // want "invalid or duplicate fallback"
	NullOnValue       string            `shape:"ifnull=x"`          // want "ifnull requires a pointer"
	ZeroOnPointer     *string           `shape:"ifzero=x"`          // want "ifzero literal is unsupported"
	NullStruct        *Child            `shape:"ifnull=x"`          // want "ifnull requires a pointer to scalar"
	NotNullValue      string            `shape:"notnull"`           // want "notnull requires pointer, slice, or map"
	NotNullArgument   *string           `shape:"notnull=x"`         // want "notnull requires pointer, slice, or map"
	NotEmptyValue     int               `shape:"notempty"`          // want "notempty requires string, pointer, slice, or map"
	NotEmptyArgument  string            `shape:"notempty=x"`        // want "notempty requires string, pointer, slice, or map"
	TrimNumber        int               `shape:"trim"`              // want "trim requires string"
	LowerArgument     string            `shape:"tolower=x"`         // want "tolower requires string and takes no value"
	LengthNumber      int               `shape:"minlength=1"`       // want "minlength requires a string value"
	EmptyPattern      string            `shape:"pattern=''"`        // want "pattern must not be empty"
	InvalidPattern    string            `shape:"pattern='['"`       // want "invalid pattern"
	LenBool           bool              `shape:"len=1"`             // want "len requires string, slice, or map"
	MinBool           bool              `shape:"min=1"`             // want "min requires number, duration, slice, or map"
	CompareString     string            `shape:"gt=1"`              // want "gt requires number or duration"
	OneOfBool         bool              `shape:"oneof=true"`        // want "oneof requires string, number, or duration"
	PositiveString    string            `shape:"positive"`          // want "positive requires number or duration"
	PositiveArgument  int               `shape:"positive=1"`        // want "positive requires number or duration and takes no value"
	UniqueMap         map[string]string `shape:"unique"`            // want "unique requires slice and takes no value"
	UniqueArgument    []string          `shape:"unique=true"`       // want "unique requires slice and takes no value"
}

type Child struct {
	Name string
}

type InvalidConstants struct {
	UnsignedNegative uint8               `shape:"min=-1"`                    // want "invalid or overflowing number"
	SignedOverflow   int64               `shape:"max=999999999999999999999"` // want "invalid or overflowing number"
	FloatHex         float64             `shape:"ifzero=0x1p2"`              // want "invalid or overflowing number"
	FloatInfinite    float32             `shape:"max=1e999"`                 // want "invalid or overflowing number"
	DurationBad      shapetypes.Duration `shape:"min=tomorrow"`              // want "invalid duration"
	TimeBad          time.Time           `shape:"ifzero=yesterday"`          // want "invalid RFC3339 time"
	CollectionBad    []string            `shape:"max=-1"`                    // want "invalid non-negative collection length"
	LengthBad        string              `shape:"len=-1"`                    // want "invalid non-negative length"
	BetweenArity     int                 `shape:"between=1"`                 // want "invalid list arity"
	BetweenReverse   int                 `shape:"between=2|1"`               // want "between minimum exceeds maximum"
	BetweenEmpty     int                 `shape:"between=1|"`                // want "empty list value"
	OneOfEmptyNumber int                 `shape:"oneof=1||2"`                // want "empty list value"
}

type ArrayField struct{ Value [2]string }
type InterfaceField struct{ Value any }
type ComplexField struct{ Value complex64 }
type FuncField struct{ Value func() }
type ChannelField struct{ Value chan int }
type UintptrField struct{ Value uintptr }
type DoublePointer struct{ Value **string }
type SlicePointer struct{ Value *[]string }
type MapPointer struct{ Value *map[string]int }
type FloatMapKey struct{ Value map[float64]int }
type Recursive struct{ Next *Recursive }
type Embedded struct{ Child }
type TaggedIgnored struct {
	Hidden string `json:"-" shape:"trim"`
	secret string `shape:"trim"`
}

type Explicit struct {
	Name    string
	Count   int
	private string
	Hidden  string `json:"-"`
	Child
}

func invalidConstruction(data []byte) {
	_ = shape.Struct[ArrayField]()     // want "unsupported field type"
	_ = shape.Struct[InterfaceField]() // want "unsupported field type"
	_ = shape.Struct[ComplexField]()   // want "unsupported field type"
	_ = shape.Struct[FuncField]()      // want "unsupported field type"
	_ = shape.Struct[ChannelField]()   // want "unsupported field type"
	_ = shape.Struct[UintptrField]()   // want "unsupported field type"
	_ = shape.Struct[DoublePointer]()  // want "unsupported pointer field type"
	_ = shape.Struct[SlicePointer]()   // want "unsupported pointer field type"
	_ = shape.Struct[MapPointer]()     // want "unsupported pointer field type"
	_ = shape.Struct[FloatMapKey]()    // want "map key must be string or integer"
	_ = shape.Struct[Recursive]()      // want "recursive type is unsupported"
	_ = shape.Struct[Embedded]()       // want "anonymous fields are unsupported"
	_ = shape.Struct[TaggedIgnored]()  // want "has a tag but is not processed"

	var invalid InterfaceField
	_ = shape.BindJSON(&invalid, data)                                                       // want "unsupported field type"
	_ = shape.BindJSONContext(context.Background(), &invalid, data)                          // want "unsupported field type"
	_ = shape.BindJSONReader(&invalid, strings.NewReader("{}"))                              // want "unsupported field type"
	_ = shape.BindJSONReaderContext(context.Background(), &invalid, strings.NewReader("{}")) // want "unsupported field type"

	_ = shape.New[Explicit](shape.String("Missing")) // want "target has no direct field Missing"
	_ = shape.New[Explicit](shape.String("private")) // want "field private is not exported"
	_ = shape.New[Explicit](shape.String("Hidden"))  // want "field Hidden is excluded from JSON"
	_ = shape.New[Explicit](shape.Int("Name"))       // want "field Name has type string, schema has type int"
	_ = shape.New[Explicit](shape.String("Count"))   // want "field Count has type int, schema has type string"
	_ = shape.New[Explicit](shape.String("Child"))   // want "target has no direct field Child"
}
