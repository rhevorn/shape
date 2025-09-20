// Package shape defines typed transformation, validation, and input binding.
//
// New defines an explicit Schema from typed Field bindings. FromTags compiles a
// cached Schema from struct tags. Schema itself is Transform + Validate only.
// StructSpec and TaggedSpec also provide ParseJSON, ParseForm, and ParseQuery.
// Package-level ParseJSON works with any Schema; BindJSON is the tag-driven
// shortcut that needs no Schema variable and writes through a pointer.
// ParseForm and ParseQuery accept url.Values for struct targets; BindForm and
// BindQuery derive the tagged Schema and replace the target only on success.
// BindRequest combines Query and a Content-Type-selected body before processing.
//
// Use the validate and transform subpackages when only one capability is needed.
package shape
