# Contributing

GoShape is in its initial design and implementation phase. Keep changes small,
dependency-light, and covered by tests.

Before submitting a change, run:

```sh
make check
make test-race
make fuzz-smoke
```

Public API changes should also update the examples and the architecture notes.
Adding an external dependency requires a concrete justification; the core
module should remain dependency-free unless there is a compelling reason.

The project is licensed under MIT. Contributions are accepted under the same
license.
