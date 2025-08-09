package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/rhevorn/shape/internal/fieldmeta"
	"github.com/rhevorn/shape/internal/jsondecode"
	"github.com/rhevorn/shape/internal/valuesdecode"
)

// RequestPrecedence controls conflicts between Query and the request body.
type RequestPrecedence uint8

const (
	// RejectConflicts rejects a Go field supplied by both Query and the body.
	RejectConflicts RequestPrecedence = iota
	// QueryFirst selects the submitted Query field, including its zero value.
	QueryFirst
	// BodyFirst selects the submitted body field, including null or a zero value.
	BodyFirst
)

// DefaultMaxRequestBytes is the body limit used by BindRequest without an explicit limit.
const DefaultMaxRequestBytes int64 = 1 << 20

var (
	// ErrRequestTooLarge reports a body larger than RequestOptions.MaxBytes.
	ErrRequestTooLarge = errors.New("shape: request body too large")
	// ErrUnsupportedContentType reports a nonempty body with no supported media type.
	ErrUnsupportedContentType = errors.New("shape: unsupported request content type")
	// ErrRequestFiles reports file parts, which must be handled by the HTTP handler.
	ErrRequestFiles = errors.New("shape: BindRequest does not accept file parts; parse multipart and use BindForm")
)

// RequestOptions controls HTTP decoding. MaxBytes zero uses DefaultMaxRequestBytes;
// a positive value sets the body limit. Negative limits and invalid precedences panic.
// Both sources must decode successfully, including fields overridden by precedence.
type RequestOptions struct {
	DisallowUnknownFields bool
	MaxBytes              int64
	Precedence            RequestPrecedence
}

// SourceConflictError identifies a top-level Go field submitted by multiple sources.
// Objects, slices, and maps are whole fields; their contents are never recursively merged.
type SourceConflictError struct {
	Field   string
	Sources []string
}

func (e *SourceConflictError) Error() string {
	if e == nil {
		return "shape: conflicting input sources"
	}
	return fmt.Sprintf("shape: field %s supplied by %s", e.Field, strings.Join(e.Sources, " and "))
}

// BindRequest decodes Query and, for a nonempty body, JSON or form fields selected
// by Content-Type. It transforms and validates the merged value once, then replaces
// *target on success. It uses request.Context and consumes request.Body without closing
// it. Call before another body parser. Multipart text is supported; file parts are not.
func BindRequest[T any](target *T, request *http.Request, options ...RequestOptions) error {
	if len(options) > 1 {
		panic("shape: at most one RequestOptions")
	}
	var opts RequestOptions
	if len(options) == 1 {
		opts = options[0]
	}
	if opts.MaxBytes < 0 {
		panic("shape: MaxBytes must not be negative")
	}
	if opts.Precedence > BodyFirst {
		panic("shape: invalid RequestPrecedence")
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = DefaultMaxRequestBytes
	}
	if target == nil {
		return errors.New("shape: nil bind target")
	}
	if request == nil || request.URL == nil {
		return errors.New("shape: nil request or request URL")
	}
	ctx := request.Context()
	if err := ctx.Err(); err != nil {
		return err
	}
	if request.Form != nil || request.PostForm != nil || request.MultipartForm != nil {
		return errors.New("shape: request form already parsed; use BindForm or BindQuery")
	}
	schema := FromTags[T]()
	typ := reflect.TypeFor[T]()
	valuesdecode.Check(typ, fieldmeta.Query)
	valuesdecode.Check(typ, fieldmeta.Form)
	query, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return fmt.Errorf("shape: decode query: %w", err)
	}
	var queryValue T
	queryOwned := true
	var queryFields []bool
	if len(query) != 0 {
		queryValue, queryOwned, err = decodeParameters[T](ctx, query, fieldmeta.Query, opts.DisallowUnknownFields)
		if err != nil {
			return err
		}
		queryFields = valuesdecode.Presence(typ, query, fieldmeta.Query)
	}
	body, err := readRequestBody(ctx, request.Body, opts.MaxBytes)
	if err != nil {
		return err
	}
	var bodyValue T
	bodyOwned := true
	var bodyFields []bool
	bodySource := ""
	if len(body) != 0 {
		mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil {
			return fmt.Errorf("%w: %s", ErrUnsupportedContentType, request.Header.Get("Content-Type"))
		}
		switch {
		case mediaType == "application/json" || strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json"):
			bodySource = "json"
			bodyValue, bodyFields, err = decodeRequestJSON[T](ctx, body, opts.DisallowUnknownFields)
			bodyOwned = jsondecode.OwnsStorage(typ)
		case mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data":
			bodySource = "form"
			var values url.Values
			if mediaType == "application/x-www-form-urlencoded" {
				values, err = url.ParseQuery(string(body))
			} else {
				values, err = readMultipartValues(ctx, body, params["boundary"])
			}
			if err != nil {
				return fmt.Errorf("shape: decode form: %w", err)
			}
			bodyValue, bodyOwned, err = decodeParameters[T](ctx, values, fieldmeta.Form, opts.DisallowUnknownFields)
			if err == nil {
				bodyFields = valuesdecode.Presence(typ, values, fieldmeta.Form)
			}
		default:
			return fmt.Errorf("%w: %s", ErrUnsupportedContentType, mediaType)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
	}
	merged := reflect.ValueOf(&bodyValue).Elem()
	queryRoot := reflect.ValueOf(&queryValue).Elem()
	for i, present := range queryFields {
		if !present {
			continue
		}
		if i < len(bodyFields) && bodyFields[i] {
			switch opts.Precedence {
			case RejectConflicts:
				return &SourceConflictError{Field: typ.Field(i).Name, Sources: []string{"query", bodySource}}
			case BodyFirst:
				continue
			}
		}
		merged.Field(i).Set(queryRoot.Field(i))
	}
	// Validation paths use the stable schema names because a request can mix inputs.
	ctx = fieldmeta.WithSource(ctx, fieldmeta.Value)
	out, err := finishParse(ctx, schema, bodyValue, queryOwned && bodyOwned)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	*target = out
	return nil
}

type requestReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r requestReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if r.ctx.Err() != nil {
		return n, r.ctx.Err()
	}
	return n, err
}

func readRequestBody(ctx context.Context, reader io.Reader, limit int64) ([]byte, error) {
	if reader == nil {
		return nil, ctx.Err()
	}
	r := requestReader{ctx, reader}
	body, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) == limit {
		extra, err := io.ReadAll(io.LimitReader(r, 1))
		if err != nil {
			return nil, err
		}
		if len(extra) != 0 {
			return nil, ErrRequestTooLarge
		}
	}
	return body, ctx.Err()
}

func readMultipartValues(ctx context.Context, body []byte, boundary string) (url.Values, error) {
	if boundary == "" {
		return nil, errors.New("missing multipart boundary")
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	values := make(url.Values)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		part, err := reader.NextPart()
		if err == io.EOF {
			return values, nil
		}
		if err != nil {
			return nil, err
		}
		disposition, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil || disposition != "form-data" || params["name"] == "" {
			return nil, errors.New("multipart part requires a form-data name")
		}
		if _, file := params["filename"]; file {
			return nil, ErrRequestFiles
		}
		text, err := io.ReadAll(requestReader{ctx, part})
		if err != nil {
			return nil, err
		}
		values.Add(params["name"], string(text))
	}
}

type requestJSONField struct {
	name  string
	index int
}

var requestJSONFields sync.Map
var requestJSONUnmarshaler = reflect.TypeFor[json.Unmarshaler]()

func decodeRequestJSON[T any](ctx context.Context, body []byte, strict bool) (T, []bool, error) {
	var zero T
	typ := reflect.TypeFor[T]()
	if typ.Implements(requestJSONUnmarshaler) || reflect.PointerTo(typ).Implements(requestJSONUnmarshaler) {
		return zero, nil, errors.New("shape: BindRequest cannot merge a root JSON unmarshaler; use BindJSON")
	}
	// Inspect presence without invoking T's custom field decoders.
	object, err := jsondecode.Decode[map[string]json.RawMessage](ctx, bytes.NewReader(body), jsondecode.Options{})
	if err != nil {
		return zero, nil, err
	}
	if object == nil {
		return zero, nil, errors.New("shape: JSON request body must be an object")
	}
	value, err := jsondecode.Decode[T](ctx, bytes.NewReader(body), jsondecode.Options{DisallowUnknownFields: strict})
	if err != nil {
		return zero, nil, err
	}
	cached, ok := requestJSONFields.Load(typ)
	if !ok {
		var fields []requestJSONField
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" {
				continue
			}
			if name := fieldmeta.Resolve(f.Name, f.Tag).JSON; name != "" {
				fields = append(fields, requestJSONField{name, i})
			}
		}
		cached, _ = requestJSONFields.LoadOrStore(typ, fields)
	}
	fields := cached.([]requestJSONField)
	present := make([]bool, typ.NumField())
	for key := range object {
		if err := ctx.Err(); err != nil {
			return zero, nil, err
		}
		index := -1
		for _, field := range fields {
			if field.name == key {
				index = field.index
				break
			}
			if index == -1 && strings.EqualFold(field.name, key) {
				index = field.index
			}
		}
		if index != -1 {
			present[index] = true
		}
	}
	return value, present, nil
}
