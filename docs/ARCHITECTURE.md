# Architecture and compatibility

This document fixes the package boundaries, execution model, ownership rules,
and compatibility policy for public releases. The exact exported inventory is
in [API.md](API.md).

## 1. Design goals

Shape optimizes for Schemas that are readable at the call site:

```go
var userSchema = shape.New[User](
	shape.String("Name").Trim().NotEmpty(),
	shape.Int("Age").Min(18),
)
```

The design keeps four properties explicit:

- Go types remain visible and are checked wherever the language permits.
- transformation and validation can run independently;
- JSON orchestration has one fixed, documented order;
- invalid configuration fails during construction, not during random traffic.

## 2. Package boundaries

```text
validate.Validator[T]
    reads T
    returns error
    never transforms T

transform.Transformer[T]
    reads T
    returns a T of the same Go type
    never validates T

shape.Schema[T]
    combines both capabilities for any supported T
    owns JSON decode and atomic bind
    adds paths for structs and nested values

types.Duration
    defines a human-readable JSON duration representation

jsonschema / openapi
    read an exportable Schema plan
    never affect runtime processing
```

The `validate` and `transform` packages do not import the root package. The root
package depends on both. Adapters depend on the root Schema interface.

## 3. Two Schema construction paths

### Explicit field Schema

```go
shape.New[T](fields...)
```

`T` must be an ordinary value struct. Every field Spec type-erases a strongly
typed Transformer and Validator into an immutable field operation. Construction
uses reflection once to resolve the Go field, verify its type, and record the
effective JSON path.

An explicit Schema:

- processes fields in `shape.New` argument order;
- ignores `shape` tags;
- leaves omitted fields unchanged and unvalidated;
- supports nested Schema, pointer, slice, and map composition;
- runs whole-struct `Apply`/`Refine` after field operations in the same phase.

### Tagged Schema

```go
shape.Struct[T]()
shape.BindJSON(&target, source)
```

The tagged compiler walks the supported Go field graph, parses `shape` tags,
and builds an immutable recursive plan. Successful and failed compilation
results are cached by `reflect.Type`, so repeated Bind calls do not reparse tags.

A tagged Schema:

- processes exported fields in Go declaration order;
- uses tags on nested value structs automatically;
- rejects unsupported graphs, anonymous fields, recursion, and duplicate JSON
  names during construction;
- is the current source for JSON Schema/OpenAPI export metadata.

Package-level Bind functions infer `T`, retrieve the cached tagged plan, and run
the same Schema JSON pipeline.

## 4. Independent phases

For an existing value:

```text
Schema.Transform(value)      → transformed copy or error
Schema.Validate(value)       → all validation issues or nil
Schema.ValidateFirst(value)  → first validation issue or nil
```

These calls are deliberately independent. `Validate` observes exactly the
supplied value; `Transform` does not decide whether the result is valid.

For JSON:

```text
read exactly one JSON value
    ↓
encoding/json decode into a fresh zero T
    ↓
run every field transform
    ↓
run whole-struct transforms
    ↓
run every field validation rule
    ↓
run whole-struct validation rules
    ↓
return T, or atomically replace the Bind target
```

Transform and validation methods may be visually interleaved on one field Spec,
but their phases are not interleaved at runtime. Within each phase, declarations
retain their order.

For tag text, `shape:"notempty,trim"` and `shape:"trim,notempty"` therefore both
trim during the transform phase and validate the transformed value afterward.

## 5. Stable traversal order

Determinism is part of the error contract:

```text
explicit struct fields     shape.New argument order
tagged struct fields       Go declaration order
field rules/callbacks      fluent method-call order within the phase
slices                     ascending index
maps                       sorted supported key order
```

`ValidateFirst` follows the same order and stops before evaluating later work.
`Validate` aggregates up to `validate.DefaultMaxIssues`; reaching the cap emits
the stable `too_many_issues` code.

## 6. Ownership and immutability

Constructed Validators, Transformers, and Schemas are immutable values. Fluent
methods return a new value and do not modify the receiver. They can be stored as
package variables and reused concurrently.

Built-in transforms do not mutate caller-owned pointer pointees, slice backing
arrays, or maps. They return new outer storage. Fallback values are snapshotted
at construction and copied for calls where the graph can be copied safely.

A transform pipeline detaches mutable input once and passes ownership of that
private working value through its built-in steps. `Then` is flattened, and
Pointer/Slice/Map traversal reuses owned storage instead of copying it at every
level. Multiple whole-struct `Apply` calls also share one detached working copy.
Fresh values produced by JSON decoding use an internal owned path, so the
runtime does not add a defensive copy where mutation is already isolated.

An `Apply` callback receives the private working value. Mutable storage returned
by that callback is treated as owned by the rest of the pipeline.

No partial value is returned after transform failure. Bind never changes the
target after a decode, transform, validation, size-limit, or context failure.

## 7. JSON boundary

Shape intentionally delegates representation rules to `encoding/json`:

- `json` field names and `json:",string"` follow the standard library;
- custom `UnmarshalJSON` methods run normally;
- `[]byte` follows standard base64 behavior;
- numeric overflow and type mismatch remain decoder errors;
- missing keys and JSON null become whatever zero/nil state the standard
  decoder produces for the Go field.

Shape does not retain raw tokens or presence bits after decoding. `IfZero`
therefore cannot distinguish missing, null, and explicit zero when they decode
to the same scalar. Pointer/slice/map nil state is preserved and may be handled
with `IfNull` or `NotNull`.

Unknown-field rejection and byte limits are opt-in through `JSONOptions`.
Exactly one top-level JSON value is accepted.

## 8. Errors

Validation returns `*validate.Error`, which contains ordered `Issue` values.
Codes and paths are stable machine-facing fields; localized messages are
human-facing.

Built-in messages use a typed `validate.Language`. Process-wide language is
atomic; context locale overrides it per request. Custom Refine errors retain the
user message and receive code `custom`.

Schema transform failures use `*shape.TransformError`, prefix nested JSON paths,
and preserve the original error through `Unwrap`. Decode errors retain the
`encoding/json` cause.

Invalid program configuration panics at construction. Invalid external input
returns an error. This distinction is intentional and stable.

## 9. Context and bounds

Context-aware entry points check cancellation between rules, fields, collection
items, copies, and map traversal. `ApplyContext` and `RefineContext` receive the
same context. A non-context callback already executing cannot be preempted.

Traversal/copy depth and issue aggregation are bounded. Reader input can be
bounded with `JSONOptions.MaxBytes`. These limits protect request handlers from
pathological input without introducing hidden goroutines.

## 10. Static analysis

Go compilation catches ordinary method/type mistakes. It cannot interpret tag
strings or connect a string literal field name to `T`.

`shapevet` supplements compilation for statically visible declarations. Runtime
construction remains the final authority because Specs may be stored in
variables or assembled dynamically.

The analyzer is optional and never imported by runtime packages.

## 11. Export boundary

JSON Schema/OpenAPI export is conservative: behavior is exported only when the
adapter can represent it without changing semantics. Unsupported transforms,
fallbacks, callbacks, custom JSON representations, and rules return
`UnsupportedSchemaError`.

The initial export contract accepts representable tagged plans. Explicit field
Schemas remain runtime-only until their metadata representation can guarantee
the same fidelity. Silent omission is forbidden.

## 12. Compatibility policy

After the first public release, the following require a major version:

- removing or renaming a documented public identifier;
- changing a public signature or generic constraint incompatibly;
- changing Decode → Transform → Validate order;
- making Transform validate or Validate transform;
- changing aggregate/fail-fast semantics or traversal order;
- changing documented zero, null, empty, path, or atomic Bind behavior;
- changing an existing stable validation code to mean something different;
- adding implicit coercion to an existing strict operation.

Minor versions may add new factories, rules, transforms, languages, issue codes,
or optional adapters when existing programs retain their behavior.

The following concepts are intentionally absent from the initial contract:

```text
implicit coercion
Optional
Default
SkipNull
Nullable
NotBlank
cross-type Transform
ordinary-value Parse or in-place Bind
```

`IfZero`, `IfNull`, pointers, and explicit custom callbacks cover the intended
cases without hiding JSON state or mixing validation with transformation.
