# Contributing

GoShape targets Go 1.24 and newer. Keep changes small, dependency-light, and
covered by tests.

Before submitting a change, run:

```sh
make check
make test-race
make fuzz-smoke
```

Public API changes should also update the examples and the architecture notes.
The root module and core adapters use only the Go standard library. Adding an
external dependency requires a concrete justification and should normally use
a separate integration module so core users do not inherit it.

Prefer generics when they preserve compile-time relationships between schema
input, output, fields, and composition. Do not replace a clear concrete fast
path with reflection merely to reduce source-code repetition. Reflection is
acceptable at adapter boundaries, such as converting a decoded JSON number
into a user-defined named numeric type, when generics alone cannot construct
the value.

The project is licensed under MIT. Contributions are accepted under the same
license.
