// Package shape defines typed transformation, validation, and JSON Schemas.
//
// New defines an explicit Schema from typed Field bindings. Struct compiles a
// cached Schema from struct tags. Schema itself is Transform + Validate only.
// StructSpec and TaggedSpec also implement JSONSchema for ParseJSON.
// Package-level ParseJSON works with any Schema; BindJSON is the tag-driven
// shortcut that needs no Schema variable and writes through a pointer.
//
// Use the validate and transform subpackages when only one capability is needed.
package shape
