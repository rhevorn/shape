// Package shape provides type-safe, composable schemas for parsing and
// validating untrusted runtime values.
//
// Prefer explicit Schema values built with Object, Fields, and related
// helpers. Struct / MustStruct optionally derive the same ObjectSchema from
// json and shape tags for simple DTOs. A schema parses an unknown input,
// applies normalization and validation, and returns a typed Go value or a
// structured ValidationError.
//
// Composite errors and named recursion have conservative default bounds.
// Collection Max rules and JSON reader limits provide additional controls for
// attacker-facing input.
package shape
