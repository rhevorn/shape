# Struct tag reference

`shape:"..."` configures tag-driven `shape.BindJSON*` and the reusable Schema
returned by `shape.Struct[T]()`. This file is the complete tag specification.

Tags are the convenience form for static DTO behavior. The recommended primary
API is the explicit `shape.New[T](...)` Schema described in [USAGE.md](USAGE.md).
Both forms use the same Transform → Validate phases.

## 1. Basic form

```go
type Request struct {
    Name    string         `json:"name" shape:"ifzero=guest,trim,notempty,maxlength=50"`
    Age     int            `json:"age" shape:"min=18,max=120"`
    Enabled *bool          `json:"enabled" shape:"ifnull=true"`
    Timeout types.Duration `json:"timeout" shape:"ifzero=30s,min=1s,max=5m"`
}

var request Request
err := shape.BindJSON(&request, data)
```

The `json` tag and `encoding/json` decide how input becomes a Go value. The
`shape` tag only sees that decoded value.

Schema keeps two plans. All tag transforms run first; then all tag rules run.
Their relative position in one tag does not interleave the two phases:

```go
Value string `shape:"notempty,trim"`
```

For JSON containing spaces, `trim` produces `""` and `notempty` then fails.
Within the transform phase and within the validation phase, items retain the
order in which they appear.

## 2. Supported Go field graph

| Go type | Behavior |
| --- | --- |
| string and named string | string transforms and rules |
| bool | zero fallback; other business rules use `validate.Refine` outside tags |
| signed/unsigned integer, float, named number | numeric rules; `uintptr` excluded |
| `types.Duration` | JSON/tag values such as `30s`; numeric duration rules |
| `time.Duration` | ordinary named `int64`; constants are nanoseconds |
| `time.Time` | RFC3339 zero fallback; no comparison tags |
| ordinary value struct | recursively process exported fields |
| `*scalar`, `*time.Time`, `*types.Duration`, `*struct` | nil handled outside; non-nil value/fields processed inside |
| `[]T` | container rules plus recursive element processing |
| `map[K]V` | container rules plus recursive value processing; K is string or integer |

Arrays, interfaces/`any`, complex numbers, funcs, channels, unsafe pointers,
anonymous fields, recursive types, `**T`, `*[]T`, and `*map[K]V` are rejected.
Use the basic `validate` and `transform` packages when a value does not fit the
tagged DTO model.

Unexported fields and `json:"-"` fields are ignored, but placing a non-empty
`shape` tag on either is an error. Duplicate effective JSON field names are an
error. Paths use the effective JSON name.

## 3. Transform tags

| Tag | Types | Effect |
| --- | --- | --- |
| `ifzero=value` | scalar, number, `time.Time`, duration | replace the Go zero value |
| `ifnull=value` | one-level pointer to a scalar/time/duration | replace nil with a new pointer |
| `trim` / `trim=' /'` | string / `*string` | trim Unicode whitespace or a character set on both sides |
| `ltrim` / `ltrim=' /'` | string / `*string` | trim the left side |
| `rtrim` / `rtrim=' /'` | string / `*string` | trim the right side |
| `tolower` | string / `*string` | Unicode lower case |
| `toupper` | string / `*string` | Unicode upper case |

`ifzero` cannot distinguish a missing JSON key, JSON `null`, and an explicit
zero value after `encoding/json` maps all three to the same Go scalar zero.
Use a pointer when presence matters. `ifnull` does not run for a non-nil pointer
that points to a zero value.

## 4. Validation tags

| Tag | Types | Pass condition |
| --- | --- | --- |
| `notnull` | pointer, slice, map | value is non-nil |
| `notempty` | string, pointer, slice, map | string/container has items; pointer is non-nil |
| `minlength=n` | string / `*string` | rune count ≥ n |
| `maxlength=n` | string / `*string` | rune count ≤ n |
| `len=n` | string, slice, map | rune/item count equals n |
| `min=n` / `max=n` | number or collection | numeric value/item count meets inclusive bound |
| `gt=n`, `gte=n`, `lt=n`, `lte=n` | number / pointer to number | comparison passes |
| `between=a\|b` | number / pointer to number | inclusive range passes |
| `positive`, `negative`, `nonnegative` | number / pointer to number | comparison with zero passes |
| `oneof=a\|b\|c` | string or number and pointer forms | equals one candidate |
| `pattern=expr` | string / `*string` | Go regexp matches |
| `startswith=s`, `endswith=s`, `contains=s` | string / `*string` | content condition passes |
| `email`, `url`, `uuid`, `ip` | string / `*string` | format condition passes |
| `unique` | slice | elements are deeply unique |
| `label=text` | every supported field | set the issue display label; path is unchanged |

`NotEmpty` does not trim. A nil pointer is empty, but a pointer to `""` is not
empty at the pointer layer; string rules on that field inspect the pointee.
For a nil pointer, inner transforms and rules are skipped.

Container tags belong to the container. They are not inherited by elements,
map keys, or map values. Element/value structs use their own field tags. Map
keys cannot be configured through a struct tag.

## 5. Constants and grammar

A tag is a comma-separated list of `name` or `name=value`. Names are ASCII
case-insensitive; lowercase is canonical. Whitespace around names, `=`, and
commas is ignored.

Single quotes preserve commas, equals signs, and surrounding spaces. Inside a
quoted value, `\\`, `\'`, `\n`, `\r`, and `\t` are supported. Empty string is
written `''`. Leading/trailing commas, empty items, unterminated quotes, and
unknown escapes are errors.

| Target | Constant syntax |
| --- | --- |
| string | unquoted simple text or a single-quoted value |
| bool | exactly `true` or `false` |
| integer | base-10 integer within the exact Go type range |
| float | finite decimal or exponent form; no NaN/Infinity/hex/underscores |
| `types.Duration` | `time.ParseDuration` text such as `30s`, or `0` |
| `time.Duration` | base-10 int64 nanoseconds |
| `time.Time` | RFC3339/RFC3339Nano |
| lengths | non-negative base-10 integer |

`between` and `oneof` separate candidates with `|`. Empty candidates are
invalid. Duplicate `label` or fallback tags are invalid.

## 6. JSON states

| Field | missing key | JSON `null` | explicit empty |
| --- | --- | --- | --- |
| scalar | Go zero | Go zero | Go zero |
| value struct | zero struct | zero struct | decoded struct |
| `*T` | nil | nil | non-nil pointer |
| `[]T` | nil | nil | non-nil empty slice for `[]` |
| `map[K]V` | nil | nil | non-nil empty map for `{}` |

Shape does not inspect raw JSON tokens after decoding. `json:",string"`, custom
`UnmarshalJSON`, base64 `[]byte`, numeric overflow, and null-to-zero behavior are
the standard library's behavior.

## 7. Errors and static checking

`shape.Struct[T]()` and `shape.BindJSON*` panic for invalid program
configuration. Input data that fails a rule returns `*validate.Error`. JSON
syntax/type failures remain decode errors.

Go itself does not understand struct-tag contents. Runtime construction is the
authoritative check. CI can optionally check statically visible fields,
`shape.Struct[T]()` calls, `shape.BindJSON*` target types, and direct explicit
`shape.New[T](...)` fields:

```sh
go install github.com/rhevorn/shape/tools/shapevet@latest
go vet -vettool="$(which shapevet)" ./...
```

There are no tags named `optional`, `default`, `skipnull`, `nullable`,
`notblank`, `coerce`, `refine`, `dive`, `union`, or `recursive`.

The frozen public API and behavior guarantees are in [API.md](API.md).
