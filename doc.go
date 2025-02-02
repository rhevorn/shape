// Package shape provides type-safe, composable schemas for parsing and
// validating untrusted runtime values.
//
// Schemas are explicit values rather than struct-tag declarations. A schema
// parses an unknown input, applies normalization and validation, and returns a
// typed Go value or a structured ValidationError.
//
// Composite errors and named recursion have conservative default bounds.
// Collection Max rules and JSON reader limits provide additional controls for
// attacker-facing input.
package shape
