# GoShape v1.0 Roadmap

GoShape will make no public release before v1.0. The repository may move quickly
while it remains unreleased, but v1.0 is cut only after the public API and its
behavioral guarantees are ready to remain compatible.

## Current baseline

The dependency-free Go 1.24+ core already covers primitives, generic numbers,
collections, typed objects, transformations, refinements, explicit coercion,
metadata, JSON parsing, JSON Schema Draft 2020-12, OpenAPI 3.1, and `net/http`.
Tuple, record, and named recursive schemas close the main composition gaps.

## Gate 1 — composition completeness

- [x] Fixed-length heterogeneous tuples
- [x] Independently parsed record keys and values
- [x] Named lazy and recursive schemas
- [x] Recursive JSON Schema `$defs` and `$ref`
- [x] Lock `Union` as at-least-one and `OneOf` as exactly-one semantics
- [x] Defer a specialized discriminated union until profiling demonstrates a
  need beyond explicit `Union` composition
- [x] Add focused examples for tuple, record, and recursive composition

Exit: common JSON-shaped inputs can be modeled without reflection, tags, or
untyped output plumbing.

## Gate 2 — API and compatibility freeze

- [ ] Audit all exported names, signatures, zero values, panic conditions, and
  error types
- [ ] Publish a concise compatibility policy for schemas, issue codes, paths,
  and generated documents
- [ ] Add compile-time API examples and behavioral contract tests
- [ ] Remove accidental aliases or ambiguous semantics before they become v1
  commitments

Exit: the exported surface is intentionally supportable under semantic
versioning.

## Gate 3 — correctness and resilience

- [x] Expand parser fuzz coverage across tuple, record, and recursive paths
- [ ] Add mutation or invariant coverage for exporter paths
- [ ] Add parser resource limits where untrusted depth or size needs a bound
- [ ] Add golden tests for JSON Schema and OpenAPI documents
- [ ] Complete cancellation, deterministic-error-order, immutability, and race
  coverage for every composite schema
- [ ] Run a repository security and denial-of-service review

Exit: `go test ./...`, `go test -race ./...`, `go vet ./...`, fuzz smoke tests,
and documented resource-safety checks all pass.

## Gate 4 — performance contract

- [x] Record primitive, collection, object, recursive, and invalid-input
  baselines with `-benchmem`
- [ ] Set regression thresholds only for stable hot paths
- [ ] Confirm strict primitive parsing remains reflection-free
- [ ] Profile before making any API-affecting optimization

Exit: v1 has published, reproducible baselines and no known pathological common
path.

## Gate 5 — documentation and release readiness

- [ ] Complete package docs and runnable examples for the full public surface
- [ ] Write migration notes for all pre-v1 API changes
- [ ] Verify README, module path, Go version, license, CI, and changelog
- [ ] Run the release checklist from a clean checkout on Go 1.24 and current Go
- [ ] Tag `v1.0.0` only after every earlier gate is complete

## Deliberate non-goals for v1

- Struct-tag or reflection-first schema definition
- Hidden coercion
- A full JSON Schema validator
- Database or network validation built into the core
- Framework dependencies in the root module
- Logging, dependency injection, code generation, or a validation DSL

Framework-specific integrations may live in separate modules later. They do not
block the dependency-free core from reaching v1.
