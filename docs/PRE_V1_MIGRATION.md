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
`$defs` key and therefore must be unique within one exported document. Parsing
now stops after 64 active calls to the same Lazy schema; use `MaxDepth(n)` for a
trusted model that needs a different bound.

## Defaults and issue bounds

`Field.Default` now accepts only deeply immutable values. Replace mutable
defaults such as maps, non-nil slices, or pointers with `DefaultFunc`, which is
invoked for every missing field. Dynamic defaults intentionally make JSON
Schema export fail with `UnsupportedSchemaError`.

Composite parsers retain at most 100 issues. If more sibling failures exist,
the last issue has code `too_many_issues`. The HTTP adapter applies the same
default and exposes `WriteValidationErrorLimit` for a smaller response cap.

## JSON readers

`ParseJSONReader` now decodes directly from the stream. Use
`ParseJSONReaderLimit` for untrusted readers; oversized inputs return
`ErrJSONTooLarge`.
