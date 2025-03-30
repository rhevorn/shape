# Changelog

## [Unreleased]

### Added

- A pure `validate.Validator[T]` API with aggregate and fail-fast calls,
  structured paths, stable issue codes, localization, composition, and
  context-aware refinements.
- A pure `transform.Transformer[T]` API with same-type fluent transforms,
  immutable composition, conditional `IfZero`/`IfNull`, and context-aware
  custom `Apply` callbacks.
- Code-defined struct schemas through typed `shape.New[T](...)` field Schemas,
  including scalar, pointer, slice, map, and nested Schema fields.
- Cached tag schemas through `shape.Struct[T]()` and inferred package-level
  `shape.BindJSON*` helpers.
- One `shape.Schema[T]` interface (Transform + Validate) implemented by scalar,
  pointer, slice, map, explicit struct, and tagged struct Specs.
- `shape.JSONSchema[T]` on `StructSpec` / `TaggedSpec` for ParseJSON, plus
  package-level `ParseJSON*` for any Schema root and tag-driven `BindJSON*`
  that writes through a pointer with no Schema variable.
- Single-working-copy transform execution, flattened composition, owned JSON
  decoding paths, and bulk copying for slices with immutable elements.
- JSON decode → transform → validate workflow, atomic package `BindJSON`,
  unknown-field rejection, byte limits, and context cancellation.
- Nested struct, pointer, slice, map, `time.Time`, and human-readable
  `types.Duration` tag support.
- JSON Schema Draft 2020-12 and OpenAPI 3.1 adapters, plus the optional
  `shapevet` analyzer.
- A Shape-first example suite plus usage, tag, and frozen public API
  documentation.

### Fixed

- Nested-Schema transform failures keep the collection index or key in the
  reported path (`Items[0].Sku`), matching what `Validate` already reported.
  An error wrapped with `%w` by user code no longer loses its inner segments.
- A whole-struct `Apply` that fails no longer leaves caller-owned storage
  mutated on the tagged path. It was already safe on the explicit path; the two
  now share one policy.
- `shape.Value[float64]` and `validate.Value[float64]` reject NaN and infinities
  the same way `Float64` and the tagged path always have.
- `Between` panics on a NaN bound instead of silently accepting every value.
- Deep `Unique` reports `unique_limit` rather than `too_big`, so a resource
  bound is no longer indistinguishable from a length violation.
- Detached working copies no longer share unexported mutable fields (a private
  slice or map, including one reached through a promoted exported field) with
  the caller.
- `IfZero` treats a zero `time.Time` carrying a non-nil `Location` as zero, so
  the fallback is applied.
- A label set before `And` reaches the appended validators' issues in every
  family; `Slice`, `Map`, and `Pointer` used to drop it.
- Map key and value failures at the same entry are distinguishable.
- `shapevet` no longer disagrees with the runtime: float map keys are rejected,
  named slice and map types are accepted, empty `oneof` candidates are accepted,
  and tags on type-parameter fields are left to the construction call.
- The exported document no longer offers a `null` branch that a pointer-level
  `notnull`/`notempty` rejects, and an optional pointer is nullable exactly once.
- A type with only `MarshalText`/`UnmarshalText` is refused for export instead
  of being described as a struct.
- The explicit Schema path rejects two fields that share a JSON name, matching
  the tagged compiler.
- The exported `*Validator` zero value reports a configuration mistake for any
  input, instead of dereferencing nil for some values and passing others.
- Slice element validation stops on cancellation, like the map loop.
- A label on a pointer field is no longer dropped by tag compilation.
- A foreign `Transformer`'s output is detached before later steps use it.

### Changed

- `And` returns the concrete family validator type instead of `Validator[T]`,
  so rule methods may follow it.
- `validate.MaxDeepUniqueItems` and `validate.CodeUniqueLimit` are exported.
- `validate.PointerValidator.NotEmpty` is documented as a synonym of `NotNull`.

### Removed

- The pre-release `Contract`, `Option`, `Rule`, `Object`, `Fields`, typed
  `Parse`, and typed `Bind` APIs.
- Ambiguous coercion, `Optional`, `Default`, `SkipNull`, `NotBlank`, union, and
  loose `any` parsing APIs.

[Unreleased]: https://github.com/rhevorn/shape/commits/HEAD
