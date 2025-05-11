package spec

import "fmt"

// Kind describes the supported tag domains independently of reflect/go/types.
type Kind uint16

const (
	String Kind = 1 << iota
	Bool
	Number
	Time
	Duration
	Pointer
	Slice
	Map
	Struct
)

type Argument uint8

const (
	NoArgument Argument = iota
	RequiredArgument
	OptionalArgument
)

// Tag is shared by runtime construction and static analysis. The table remains
// private so callers cannot mutate the language definition.
type Tag struct {
	Kinds       Kind
	Argument    Argument
	Outer       bool
	Requirement string
}

var tags = map[string]Tag{
	"label":       {String | Bool | Number | Time | Duration | Pointer | Slice | Map | Struct, RequiredArgument, true, "label requires one value and may appear once"},
	"ifzero":      {String | Bool | Number | Time | Duration, RequiredArgument, true, "ifzero literal is unsupported for this type"},
	"ifnull":      {Pointer, RequiredArgument, true, "ifnull requires a pointer to scalar, time, or duration"},
	"notnull":     {Pointer | Slice | Map, NoArgument, true, "notnull requires pointer, slice, or map and takes no value"},
	"notempty":    {String | Slice | Map, NoArgument, true, "notempty requires string, slice, or map and takes no value; use notnull on pointers"},
	"trim":        {String, OptionalArgument, false, "trim requires string"},
	"ltrim":       {String, OptionalArgument, false, "ltrim requires string"},
	"rtrim":       {String, OptionalArgument, false, "rtrim requires string"},
	"tolower":     {String, NoArgument, false, "tolower requires string and takes no value"},
	"toupper":     {String, NoArgument, false, "toupper requires string and takes no value"},
	"minlength":   {String, RequiredArgument, false, "minlength requires a string value"},
	"maxlength":   {String, RequiredArgument, false, "maxlength requires a string value"},
	"len":         {String | Slice | Map, RequiredArgument, false, "len requires string, slice, or map"},
	"pattern":     {String, RequiredArgument, false, "pattern requires a string value"},
	"startswith":  {String, RequiredArgument, false, "startswith requires a string value"},
	"endswith":    {String, RequiredArgument, false, "endswith requires a string value"},
	"contains":    {String, RequiredArgument, false, "contains requires a string value"},
	"email":       {String, NoArgument, false, "email requires string and takes no value"},
	"url":         {String, NoArgument, false, "url requires string and takes no value"},
	"uuid":        {String, NoArgument, false, "uuid requires string and takes no value"},
	"ip":          {String, NoArgument, false, "ip requires string and takes no value"},
	"min":         {Number | Duration | Slice | Map, RequiredArgument, false, "min requires number, duration, slice, or map"},
	"max":         {Number | Duration | Slice | Map, RequiredArgument, false, "max requires number, duration, slice, or map"},
	"gt":          {Number | Duration, RequiredArgument, false, "gt requires number or duration"},
	"gte":         {Number | Duration, RequiredArgument, false, "gte requires number or duration"},
	"lt":          {Number | Duration, RequiredArgument, false, "lt requires number or duration"},
	"lte":         {Number | Duration, RequiredArgument, false, "lte requires number or duration"},
	"between":     {Number | Duration, RequiredArgument, false, "between requires number or duration"},
	"oneof":       {String | Number | Duration, RequiredArgument, false, "oneof requires string, number, or duration"},
	"positive":    {Number | Duration, NoArgument, false, "positive requires number or duration and takes no value"},
	"negative":    {Number | Duration, NoArgument, false, "negative requires number or duration and takes no value"},
	"nonnegative": {Number | Duration, NoArgument, false, "nonnegative requires number or duration and takes no value"},
	"unique":      {Slice, NoArgument, false, "unique requires slice and takes no value"},
}

func LookupTag(name string) (Tag, bool) { definition, ok := tags[name]; return definition, ok }

func CheckTag(name string, field, element Kind, hasArgument bool) error {
	d, ok := LookupTag(name)
	if !ok {
		return fmt.Errorf("unknown option %s", name)
	}
	target := field
	if field == Pointer && !d.Outer {
		target = element
	}
	if name == "ifnull" && element&(String|Bool|Number|Time|Duration) == 0 {
		return fmt.Errorf("%s", d.Requirement)
	}
	if d.Kinds&target == 0 || d.Argument == NoArgument && hasArgument || d.Argument == RequiredArgument && !hasArgument {
		return fmt.Errorf("%s", d.Requirement)
	}
	return nil
}
