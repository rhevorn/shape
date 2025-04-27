# Runnable examples

Every directory is a complete program. Run one from the repository root:

```sh
go run ./examples/shape/explicit
```

Start with the `shape` examples. The standalone `validate` and `transform`
packages are useful when a reusable struct Schema is unnecessary.

## Recommended `shape` API

| Example | What it shows |
| --- | --- |
| [`shape/explicit`](shape/explicit) | `shape.New`, every field family, nested Schemas, typed Transform/Validate, `Apply`, `Refine`, and JSON |
| [`shape/scalars`](shape/scalars) | String, number, bool, time, duration, named-number, and arbitrary-value Specs |
| [`shape/collections`](shape/collections) | Pointer, slice, map, inner Schemas, fallbacks, container rules, and fluent `.Pointer()` / `.Slice()` |
| [`shape/tags`](shape/tags) | Transform, validation, fallback, format, numeric, collection, nested, and label tags |
| [`shape/json`](shape/json) | Every ParseJSON/BindJSON byte, reader, context, and package-level form; strict input and size limits |
| [`shape/errors`](shape/errors) | Aggregate/fail-fast validation errors, issue fields, paths, transform errors, and locales |
| [`shape/context`](shape/context) | Context-aware transforms/rules, cancellation, and request-local language |
| [`shape/export`](shape/export) | Low-level document export, JSON Schema, OpenAPI schema/request/response, and unsupported behavior |

## Independent packages

| Example | What it shows |
| --- | --- |
| [`validate`](validate) | Every validator family, built-in rule group, `Refine`, `And`, labels, aggregate/fail-fast, and context |
| [`transform`](transform) | Every transformer family, `IfZero`, `IfNull`, string transforms, custom steps, `Then`, and context |

## Application-style examples

| Example | What it shows |
| --- | --- |
| [`config`](config) | Defaults, strict config loading, duration strings, and atomic replacement |
| [`http`](http) | Binding an HTTP request body with cancellation, strict JSON, limits, and typed error responses |

The examples intentionally print transformed values or inspected errors so the
effect of each operation is visible. The complete method inventory and frozen
semantics remain in [`docs/API.md`](../docs/API.md); the complete tag grammar is
in [`docs/TAGS.md`](../docs/TAGS.md).
