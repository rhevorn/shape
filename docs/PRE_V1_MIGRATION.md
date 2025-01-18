# Pre-v1 Migration Notes

GoShape has not published a release yet. These notes help users who have pinned
an earlier repository commit move to the current v1 candidate API.

## Go toolchain

The minimum version is Go 1.24. The root module remains standard-library-only.

## `Union` and `OneOf`

Earlier code treated `OneOf` as an alias of `Union`. The v1 candidate gives the
names their JSON Schema meanings:

- `Union` succeeds when at least one alternative matches and returns the first
  successful result.
- `OneOf` succeeds only when exactly one alternative matches.

Use `Union` when alternatives can overlap or when first-match behavior is
intended.

## Generic numbers

Use `Number[T]` and `CoerceNumber[T]` for built-in or named signed, unsigned,
and floating-point types. The concrete `Int`, `Int64`, and `Float64` builders
remain optimized convenience APIs.

## Recursive schemas

Define recursion with `Lazy(name, provider)`. The name becomes a JSON Schema
`$defs` key and therefore must be unique within one exported document.

## JSON readers

`ParseJSONReader` now decodes directly from the stream. Use
`ParseJSONReaderLimit` for untrusted readers; oversized inputs return
`ErrJSONTooLarge`.
