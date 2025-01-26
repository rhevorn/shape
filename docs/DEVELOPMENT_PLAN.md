# GoShape Initial Development Plan

Implementation status: the original v0.1 scope and the self-contained v0.2–v0.4
extensions are complete. This is retained as historical design context; current
work and release gates are tracked in [the v1 roadmap](V1_ROADMAP.md).

## Outcome

v0.1 is complete when an object schema can parse untrusted input into a typed
Go struct, normalize fields, validate field and object rules, reject unknown
fields in strict mode, and return all relevant issues with structured paths.
The v1 hardening pass later bounded retained sibling issues and marks truncation
explicitly with `too_many_issues`.

The scope in the product brief's “MVP Scope” section is authoritative when an
earlier section conflicts with it. In particular, `Time`, `Duration`, and the
`Coerce*` constructors are v0.2 work.

## Milestone 0 — contract and repository baseline

Deliverables:

- Public API contract for schemas, paths, issues, and validation errors
- Explicit behavioral decisions for JSON numbers, missing/default fields,
  refinements, transforms, and issue aggregation
- Go module, README, contribution commands, and CI-ready checks
- No runtime dependencies
- Go 1.24+ and generic-first composition

Exit criteria:

- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass
- API examples compile as tests
- Every unresolved public-API question is recorded rather than silently guessed

## Milestone 1 — core and primitive schemas

Implement:

- `Schema[T]`, `Parse`, and `ParseContext`
- `Issue`, `ValidationError`, `Path`, and stable error-code constants
- `String`, `Int`, `Int64`, `Float64`, and `Bool`
- String normalization: `Trim`
- String rules: `Min`, `Max`, `Len`, `Pattern`, and `Email`
- Numeric rules: `Min`, `Max`, `Gt`, `Gte`, `Lt`, and `Lte`
- Value-level `Refine`

Test focus:

- Exact boundary behavior and invalid input types
- Unicode length policy for strings
- Schema branching/immutability and concurrent reuse
- Stable error codes, expected/received values, and root paths

Exit criteria:

- Primitive behavior is table-tested
- Race tests demonstrate safe reuse of constructed schemas
- No reflection is used in primitive parse paths

## Milestone 2 — composition

Implement:

- `Slice`, including `Min` and `Max`
- `Map` with string keys and typed values
- `Transform[A, B]`
- Nested error-path prefixing and sibling issue aggregation

Test focus:

- Index and field paths such as `users[3].address.zip`
- Multiple invalid elements in one collection
- Transform errors mapped to `transform_failed`
- Context propagation and cancellation

Exit criteria:

- Nested collections preserve full paths
- Schemas remain immutable when derived from a shared base

## Milestone 3 — typed objects

Implement:

- `Object[T]` and strongly typed `Field[T, V]`
- Required-by-default fields
- `Optional` and `Default`
- Default strip mode and opt-in `Strict`
- Object-level `Refine`
- Aggregation of field, unknown-field, and object-level issues

Test focus:

- Heterogeneous field value types
- Setter invocation rules
- Missing, optional, defaulted, and present-but-invalid inputs
- Strict versus strip behavior
- Cross-field validation
- Deeply nested field/index paths

Exit criteria:

- The complete `UserSchema` definition-of-done example compiles and passes
- Invalid email, low age, and unknown input produce separate structured issues

## Milestone 4 — JSON, hardening, and release

Implement:

- `ParseJSON` using `json.Decoder.UseNumber`
- Rejection of trailing JSON values
- Package examples and API documentation
- Benchmarks for primitives, slices, and valid/invalid nested objects
- Fuzz targets for number decoding, object input, and JSON parsing

Release checks:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- Fuzz smoke run
- Benchmarks recorded as an initial baseline
- README examples compile
- MIT license present
- Canonical module path confirmed as `github.com/rhevorn/shape`

## Suggested implementation order

Work vertically in small reviewable changes:

1. Errors and paths
2. Schema contract and string schema
3. Numeric and bool schemas
4. Refinement and transform
5. Slice and map schemas
6. Object fields and object parsing
7. JSON adapter
8. Fuzzing, benchmarks, docs, and release polish

Each change should include tests and documentation for its public behavior. Avoid
landing all schema types in one large change.

## Deferred from v0.1

The following were not v0.1 work: unions, nullable values, enums, literals,
coercion constructors, time and duration schemas, URL/UUID/IP schemas, context
refinements, JSON Schema, OpenAPI, struct tags, integrations, i18n, code
generation, recursive schemas, and framework adapters. The self-contained
items in this list have since been implemented; see the README for the current
feature surface.
