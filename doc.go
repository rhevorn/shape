// Package goshape provides type-safe, composable schemas for parsing and
// validating untrusted runtime values.
//
// GoShape schemas are explicit values rather than struct-tag declarations. A
// schema parses an unknown input, applies normalization and validation, and
// returns a typed Go value or a structured ValidationError.
package goshape
