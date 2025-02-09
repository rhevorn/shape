# Examples

Primary path is an explicit schema (`Object` / `Fields`). Struct tags are
optional and live in their own sample.

| Dir | Shows |
| --- | --- |
| [`basic`](basic) | Explicit `Fields` + `Object` + `Parse` |
| [`json`](json) | Same with labels and invalid-input issues |
| [`http`](http) | Explicit schema + `ParseReaderLimitContext` |
| [`locale`](locale) | `SetLanguage("zh-CN")` with labeled fields |
| [`scalar`](scalar) | Scalars, `Transform`, `Slice` |
| [`config`](config) | Coerce helpers for env-style input |
| [`structtag`](structtag) | Optional `MustStruct` / `Bind` from `shape` tags |

```sh
go run ./examples/basic
go run ./examples/json
go run ./examples/locale
go run ./examples/scalar
go run ./examples/config
go run ./examples/structtag
go run ./examples/http   # listens on :8080
```
