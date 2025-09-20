# Forms and query parameters

Form and query parsing accepts `net/url.Values` and runs decode → transform →
validate. Both inputs reuse the same Schema as JSON.

```go
type User struct {
    Name string   `json:"name" form:"name" query:"name" shape:"trim,notempty"`
    Age  int      `json:"age" form:"age" query:"age" shape:"min=18"`
    Tags []string `json:"tags" form:"tags" query:"tags"`
}

userSchema := shape.FromTags[User]()
values := url.Values{
    "name": {" Pong "},
    "age":  {"20"},
    "tags": {"go", "web"},
}

user, err := userSchema.ParseForm(values)
user, err = shape.ParseForm(userSchema, values)

user, err = userSchema.ParseQuery(values)
user, err = shape.ParseQuery(userSchema, values)
```

Schemas created with `shape.New` provide the same methods. Package functions
accept any `Schema[T]`, but `T` must be an ordinary value struct for parameters.
Tag-driven binding needs no Schema variable:

```go
var user User
err := shape.BindForm(&user, values)
err = shape.BindQuery(&user, values)
```

Every entry point has a `Context` version with `context.Context` first. Parse
returns the zero `T` on failure. Bind replaces the entire target only after
success; omitted fields do not preserve their previous values. The caller must
not mutate the input map or its slices while parsing. Returned slices are
independent of the input slices.

## Field names and rules

| Input | Name selection |
| --- | --- |
| JSON | Standard `encoding/json` behavior |
| Form | `form` tag, or Go field name |
| Query | `query` tag, or Go field name |

The tags are independent. Form and Query names are case-sensitive; neither
falls back to `json` or to the other input. Missing and empty tags both use the
Go name. Each `"-"` tag excludes decoding from that source only. For example,
`json:"-"` does not disable Form or Query.

```go
type Request struct {
    Name string `json:"name" form:"full_name" query:"q" shape:"trim,notempty"`
    Age  int    `json:"age" form:"age" query:"age" shape:"min=18"`
}
```

Names cannot contain `.`, `[`, `]`, or `,`; dots separate nested fields, brackets
identify indexes in error paths, and form/query tags have no comma-separated
options. Duplicate names are rejected. Schema construction also rejects
ambiguous paths for direct Transform/Validate.

A Schema's field rules are fixed at construction. Input exclusion never skips
a fallback, transform, or validation rule: an excluded or absent field starts
at zero and runs its normal rules. For example, a field tagged
`json:"-" shape:"notempty"` remains required by that Schema, so JSON parsing
cannot satisfy it without an explicit fallback or whole-struct transform.
Use separate DTOs or explicitly composed Schemas when endpoints need different
rules.

`FromTags` processes every exported field in its supported type graph,
including fields excluded from all input formats. `New` processes its explicitly
bound fields. Unexported fields are not processed. Whole-struct callbacks run
after field behavior. Parsing and independently transforming then validating
use the same rules; user callbacks that depend on changing external state can
still give different results on repeated calls.

ParseForm/ParseQuery errors use that input's field names, falling back to the
Schema name for excluded fields. Direct Transform/Validate and BindRequest use
stable Schema names: effective JSON names, or Go names when JSON is disabled.
Custom callbacks supply their own paths. JSON Schema export rejects
non-JSON fields whose transforms or zero-rejecting rules cannot be represented;
it never silently drops those constraints.

## Decoding rules

| Input | Result |
| --- | --- |
| Missing field | Zero value or nil, then normal fallbacks and rules |
| `name=` into string | Empty string |
| `age=` into number | Decoding error |
| Repeated scalar parameter | Decoding error, even when the values are equal |
| `tags=a&tags=b` into a slice | Two elements in input order |
| `tags=a,b` into `[]string` | One element, `"a,b"` |
| Map entry with nil or empty value slice | Decoding error |
| Integer | Base 10, checked against the destination type's range |
| Float | Decimal syntax, including exponents; finite and within range |
| Boolean | Exactly `true`, `false`, `1`, or `0` |
| Unknown parameter | Ignored unless `DisallowUnknownFields` is enabled |

There is no implicit whitespace trimming, comma splitting, base64 decoding, or
`null` literal. A missing `*string` remains nil; an explicitly empty value
creates a pointer to `""`. Numeric parsing happens before schema transforms.
For an HTML checkbox, use `value="true"`; an unchecked checkbox is absent.

Supported fields are strings, booleans, signed and unsigned integers except
`uintptr`, floats, named scalar types, pointers to scalars or structs, scalar
slices (including scalar pointers), and nested structs. `[]byte` is a sequence
of decimal byte values. `time.Time` uses RFC3339 and `types.Duration` accepts
duration text such as `30s`. Ordinary `time.Duration` uses integer nanoseconds.

Types implementing `encoding.TextUnmarshaler` are decoded from one text value.
A custom text decoder takes precedence over built-in scalar or slice decoding.
Such types must also satisfy the chosen Schema's normal type restrictions;
`FromTags` still checks its supported field graph.

Nested structs use dot-separated keys, such as `address.city`. A nested pointer
is allocated only when one of its known descendant fields is supplied. Sending
`address=value` is an error; a struct is not a text field unless it implements
`encoding.TextUnmarshaler`.

Object slices, ordinary maps, arrays, interfaces, anonymous fields, recursive
structs, and multi-level pointers are not built-in parameter formats. Exclude
such fields from the input, use a suitable custom text type, or use JSON for
structured data. Indexed keys such as `items[0].name` and aliases such as
`tags[]` are not recognized; strict decoding rejects them.

Invalid input definitions panic when the input's cached decoding plan is first
used. This happens even when the unsupported field is absent. `shapevet` also
checks statically visible Form/Query Parse and Bind calls. Definition checks
and parameter decoding follow declaration order; slice values follow input
order. After known fields decode, malformed group keys and, in strict mode,
unknown keys are checked in lexical order. Transform and validation retain the
Schema's normal traversal order.

## Options and errors

```go
user, err := userSchema.ParseForm(values, shape.FormOptions{
    DisallowUnknownFields: true,
})
user, err = userSchema.ParseQuery(values, shape.QueryOptions{
    DisallowUnknownFields: true,
})
```

Pass at most one options value. `*shape.ParameterError` identifies decoding
failures through `Source` (`"form"` or `"query"`), `Path`, and `Err`; it supports
`errors.As` and `errors.Is`. Transform failures use `*shape.TransformError`,
validation failures use `*validate.Error`, and cancellation returns the context
error. No partial value is returned.

## HTTP integration

For query strings, check URL decoding errors before using the values:

```go
values, err := url.ParseQuery(r.URL.RawQuery)
if err != nil {
    // Return a bad-request response.
    return
}
user, err := userSchema.ParseQueryContext(r.Context(), values)
```

`r.URL.Query()` discards malformed query-string errors. Shape receives already
parsed values and cannot recover those errors.

For `application/x-www-form-urlencoded`, bound the body and pass `PostForm`:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
if err := r.ParseForm(); err != nil {
    // Return a bad-request or request-too-large response.
    return
}
err := shape.BindFormContext(r.Context(), &user, r.PostForm)
```

`r.Form` merges query and body parameters. Use `r.PostForm` for body-only
binding. A GET request's query parameters belong in ParseQuery/BindQuery.

For `multipart/form-data`, use the text values and clean up temporary files:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
err := r.ParseMultipartForm(64 << 10)
if r.MultipartForm != nil {
    defer r.MultipartForm.RemoveAll()
}
if err != nil {
    // Return a bad-request or request-too-large response.
    return
}
err = shape.BindFormContext(r.Context(), &user, r.MultipartForm.Value)
```

The `ParseMultipartForm` argument is a memory threshold, not a request-size
limit. File parts remain in `r.MultipartForm.File`; Shape does not bind or
validate their contents. HTTP parsing, request-size limits, and file handling
remain the handler's responsibility. See the runnable
[HTTP parameter example](../examples/http-parameters).

## Automatic HTTP binding

```go
type Request struct {
    Name string `json:"name" form:"full_name" query:"q" shape:"trim,notempty"`
    Age  int    `json:"age" form:"age" query:"years" shape:"min=18"`
}

var request Request
err := shape.BindRequest(&request, r, shape.RequestOptions{
    DisallowUnknownFields: true,
})
```

BindRequest parses Query, decodes the body according to Content-Type, combines
submitted fields, then transforms and validates once. A JSON body containing
`{"name":"Pong"}` and a query containing `years=20` together satisfy the Schema.
It uses `r.Context()`; there is no separate context argument. Like the other
Bind functions, it derives `FromTags[T]()` and replaces the entire target only
after success. It is not a PATCH operation.

Supported nonempty bodies are `application/json`, `application/*+json`,
`application/x-www-form-urlencoded`, and multipart text fields. An empty body
uses Query alone; a nonempty body requires a supported Content-Type. Invalid
JSON never falls back to form decoding. JSON request bodies must be objects;
root custom JSON unmarshaling requires the explicit BindJSON entry point.
Custom field decoders remain supported.

Each input uses its own tags. Fields are combined by **top-level Go field**,
not by parameter spelling. By default, submitting a field in both Query and
the body returns `*shape.SourceConflictError`, even if the two values agree.
Conflicts are reported in Go declaration order. The error identifies the Go
`Field` and its `Sources` (`query` plus `json` or `form`).

Explicit precedence is available when the endpoint needs it:

```go
err := shape.BindRequest(&request, r, shape.RequestOptions{
    Precedence: shape.QueryFirst, // or shape.BodyFirst
})
```

The zero precedence is `shape.RejectConflicts`. Presence, not the Go zero
value, chooses a winner: `0`, `false`, `""`, empty arrays, and JSON `null` all
count as submitted. Both sources must decode successfully, even if one would
be overridden. A malformed high-priority value never falls back to another
source.

Objects, pointers, slices, and maps are whole top-level fields. For example,
Query `address.city=x` and body `{"address":{"zip":"10000"}}` conflict on
`Address`. QueryFirst selects the entire Query address; it does not retain the
body's ZIP code. Unknown parameters ignored in non-strict mode do not establish
field presence. BindRequest checks and caches both Form and Query decoding
plans before processing input. Fields unsupported by those formats must have
`form:"-"` / `query:"-"` tags, even for JSON-only requests. `shapevet` checks
both plans at statically visible BindRequest calls.

BindRequest consumes `r.Body` without closing it. Call it before another body
parser; pre-parsed Form/PostForm/MultipartForm is rejected. It reads the raw
request and never mixes `r.Form` with the body. Multipart binding creates no
temporary files and rejects file parts with `ErrRequestFiles`. To handle uploads,
parse multipart in the handler, manage its files, and pass its text values to
BindForm as shown above.

`RequestOptions.MaxBytes` defaults to `DefaultMaxRequestBytes` (1 MiB) when zero;
a positive value overrides that body limit. This differs from JSONOptions,
whose zero limit is unlimited. Exceeding the limit returns `ErrRequestTooLarge`.
Query size is governed by the HTTP server's URL/header limits. Nonempty bodies
with unsupported media types return `ErrUnsupportedContentType`. Invalid URL
encoding, multipart, JSON, and parameter data return decoding errors. Negative
limits, unknown precedence values, or multiple options values panic.
