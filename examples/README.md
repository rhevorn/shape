# Examples

Shape is the recommended entry point, so the examples start with complete
struct Schemas. Run any entry from the module root:

```sh
go run ./examples/shape/explicit
```

## Recommended: shape

| Example | Demonstrates |
| --- | --- |
| [shape/explicit](shape/explicit) | Explicit fields, composites, nested Schema, `Apply`, `Refine`, and `ParseJSON` |
| [shape/types](shape/types) | Every scalar factory, named numbers, `time.Time`, `types.Duration`, and `Value[T]` |
| [shape/tags](shape/tags) | Cached `Struct[T]` and inferred tag binding |
| [shape/json](shape/json) | Reader input, strict JSON, byte limits, and atomic Bind |
| [shape/errors](shape/errors) | Aggregate/fail-fast issues, paths, labels, and locale |
| [shape/export](shape/export) | JSON Schema Draft 2020-12 and OpenAPI 3.1 export |

## Independent building blocks

| Example | Demonstrates |
| --- | --- |
| [validate](validate) | Values, scalars, pointers, slices, maps, custom rules, composition, and fail-fast |
| [transform](transform) | The same type families, fallbacks, custom steps, and composition |

## Integration

| Example | Demonstrates |
| --- | --- |
| [config](config) | Tagged defaults and human-readable duration strings |
| [http](http) | Request cancellation, reader limits, and strict JSON |

The independent examples intentionally run without importing the root package.
The root `shape` package combines both capabilities for complete JSON struct
contracts.
