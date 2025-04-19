# Contributing

`shape` targets Go 1.24 and newer. Keep `go.mod`, CI, examples, and
documentation in sync with this minimum. Keep changes small, dependency-light,
and covered by tests.

Before submitting a change, run:

```sh
make release-check
```

`tools/shapevet` is a separate Go module so its `golang.org/x/tools`
dependency does not enter the library module graph. For a release, publish the
root module tag first, then publish the matching nested-module tag (for
example, `v0.1.0` followed by `tools/shapevet/v0.1.0`). Update the root version
required by `tools/shapevet/go.mod` when the two modules move together.

Public API changes must update `docs/API.md` and relevant examples. After the
first release, follow the compatibility policy in `docs/API.md`; incompatible
behavior requires a major version.
Runtime packages and core adapters use only the Go standard library. The
`shapevet` command uses `golang.org/x/tools`; runtime packages must not import
it. Any additional dependency requires a concrete justification.

Prefer generics when they preserve compile-time relationships between Schema
input, fields, and composition. Reflection is limited to resolving explicit
fields, compiling tagged plans, and executing precompiled field operations;
JSON always decodes directly into the Schema target type.

Schema builders are immutable values: copy internal slices and maps before any write,
and keep constructed schemas safe for concurrent reuse. Every public behavior
change needs tests (`go test ./...`, `go test -race ./...`, `go vet ./...`,
plus relevant fuzz or benchmark checks).

Keep the core API explicit. Do not add hidden coercion, logging, or network
behavior. The root package owns explicit and tagged struct/JSON Schemas;
ordinary typed values also remain independently supported by `validate` and
`transform`. Tag names must map directly to one of those basic capabilities.
Process-wide language and per-request locale belong to `validate`.

## Source layout

The root package is a public facade: keep constructors, Schema types, field
Specs, JSON entry points, and export entry points there. Implementation belongs
under `internal`: `tagged` compiles and executes struct-tag plans, `program`
executes explicit fields, `pipeline` runs whole-value transforms, and
`jsondecode` owns the standard JSON decoding stage. The `validate`, `transform`,
`jsonschema`, `openapi`, and `types` directories are intentionally public
packages; do not move root implementation into a new public helper package.

The project is licensed under MIT. Contributions are accepted under the same
license.
