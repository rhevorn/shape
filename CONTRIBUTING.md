# Contributing

`shape` targets Go 1.24 and newer. Keep `go.mod`, CI, examples, and
documentation in sync with this minimum. Keep changes small, dependency-light,
and covered by tests.

Before submitting a change, run:

```sh
make release-check
```

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

The project is licensed under MIT. Contributions are accepted under the same
license.
