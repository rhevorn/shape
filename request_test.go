package shape_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/rhevorn/shape"
)

func parameterRequest(rawQuery, contentType, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/?"+rawQuery, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

type requestUser struct {
	Name string   `json:"name" form:"full_name" query:"q" shape:"trim,notempty"`
	Age  int      `json:"age" form:"age" query:"years" shape:"min=18"`
	Tags []string `json:"tags" form:"tag" query:"tag"`
}

func TestBindRequestMergesBeforeValidation(t *testing.T) {
	for _, tc := range []struct{ query, media, body string }{
		{"years=20", "application/json", `{"name":" Pong ","tags":["a","b"]}`},
		{"years=20", "application/vnd.example+json; charset=utf-8", `{"name":" Pong ","tags":["a","b"]}`},
		{"years=20", "application/x-www-form-urlencoded", "full_name=+Pong+&tag=a&tag=b"},
		{"years=20&q=+Pong+&tag=a&tag=b", "", ""},
		{"", "application/json", `{"name":" Pong ","age":20,"tags":["a","b"]}`},
	} {
		var out requestUser
		err := shape.BindRequest(&out, parameterRequest(tc.query, tc.media, tc.body), shape.RequestOptions{DisallowUnknownFields: true})
		if err != nil || out.Name != "Pong" || out.Age != 20 || !reflect.DeepEqual(out.Tags, []string{"a", "b"}) {
			t.Fatalf("out=%#v err=%v", out, err)
		}
		if err := shape.FromTags[requestUser]().Validate(out); err != nil {
			t.Fatal(err)
		}
	}
	type Once struct {
		Name string `query:"q" shape:"ifzero=guest,trim"`
	}
	var out Once
	if err := shape.BindRequest(&out, parameterRequest("q=+", "", "")); err != nil || out.Name != "" {
		t.Fatalf("transform ran more than once: %#v %v", out, err)
	}
}

func TestBindRequestSourceConflicts(t *testing.T) {
	for _, tc := range []struct{ media, body string }{
		{"application/json", `{"name":"body","age":20}`},
		{"application/json", `{"NAME":"body","age":20}`},
		{"application/x-www-form-urlencoded", "full_name=body&age=20"},
	} {
		original := requestUser{Name: "original", Age: 30, Tags: []string{"old"}}
		out := original
		err := shape.BindRequest(&out, parameterRequest("q=query", tc.media, tc.body))
		var conflict *shape.SourceConflictError
		if !errors.As(err, &conflict) || conflict.Field != "Name" || len(conflict.Sources) != 2 || !reflect.DeepEqual(out, original) {
			t.Fatalf("out=%#v err=%v", out, err)
		}
		for _, tc2 := range []struct {
			precedence shape.RequestPrecedence
			name       string
		}{{shape.QueryFirst, "query"}, {shape.BodyFirst, "body"}} {
			err := shape.BindRequest(&out, parameterRequest("q=query", tc.media, tc.body), shape.RequestOptions{Precedence: tc2.precedence})
			if err != nil || out.Name != tc2.name || out.Age != 20 || out.Tags != nil {
				t.Fatalf("out=%#v err=%v", out, err)
			}
		}
	}
}

func TestBindRequestPresenceIncludesZeroNullAndEmpty(t *testing.T) {
	type Request struct {
		Count   int      `json:"count" query:"n"`
		Enabled bool     `json:"enabled" query:"b"`
		Text    string   `json:"text" query:"s"`
		Pointer *string  `json:"pointer" query:"p"`
		List    []string `json:"list" query:"l"`
	}
	for _, tc := range []struct {
		query, body string
		precedence  shape.RequestPrecedence
		want        Request
	}{
		{"n=0&b=false&s=", `{"count":5,"enabled":true,"text":"body"}`, shape.QueryFirst, Request{}},
		{"n=5&b=true&s=query&p=x&l=x", `{"count":0,"enabled":false,"text":"","pointer":null,"list":[]}`, shape.BodyFirst, Request{List: []string{}}},
		{"n=5", `{}`, shape.BodyFirst, Request{Count: 5}},
		{"", `{"count":5}`, shape.QueryFirst, Request{Count: 5}},
	} {
		var out Request
		err := shape.BindRequest(&out, parameterRequest(tc.query, "application/json", tc.body), shape.RequestOptions{Precedence: tc.precedence})
		if err != nil || !reflect.DeepEqual(out, tc.want) {
			t.Fatalf("out=%#v want=%#v err=%v", out, tc.want, err)
		}
	}
	var out Request
	var conflict *shape.SourceConflictError
	if err := shape.BindRequest(&out, parameterRequest("p=", "application/json", `{"pointer":null}`)); !errors.As(err, &conflict) {
		t.Fatalf("null was not submitted: %v", err)
	}
}

func TestBindRequestNestedFieldsAreWholeValues(t *testing.T) {
	type Address struct {
		City string `json:"city" query:"city"`
		ZIP  string `json:"zip" query:"zip"`
	}
	type Request struct {
		Address *Address `json:"address" query:"home"`
	}
	var out Request
	query, body := "home.city=query", `{"address":{"zip":"10000"}}`
	var conflict *shape.SourceConflictError
	if err := shape.BindRequest(&out, parameterRequest(query, "application/json", body)); !errors.As(err, &conflict) || conflict.Field != "Address" {
		t.Fatal(err)
	}
	if err := shape.BindRequest(&out, parameterRequest(query, "application/json", body), shape.RequestOptions{Precedence: shape.QueryFirst}); err != nil || out.Address.City != "query" || out.Address.ZIP != "" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	if err := shape.BindRequest(&out, parameterRequest(query, "application/json", body), shape.RequestOptions{Precedence: shape.BodyFirst}); err != nil || out.Address.City != "" || out.Address.ZIP != "10000" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestBindRequestInvalidInputIsAtomic(t *testing.T) {
	for _, tc := range []struct {
		query, media, body string
		strict             bool
	}{
		{"years=bad", "application/json", `{"name":"ok","age":20}`, false},
		{"years=20", "application/json", `{"name":"ok","age":"bad"}`, false},
		{"years=20", "application/json", `{"name":`, false},
		{"years=20", "application/json", `{"name":"ok"} {}`, false},
		{"years=20", "application/json", `null`, false},
		{"years=20", "application/json", `[]`, false},
		{"q=%GG", "application/json", `{"name":"ok","age":20}`, false},
		{"q=ok;years=20", "", "", false},
		{"years=20", "application/x-www-form-urlencoded", "full_name=%GG", false},
		{"years=20", "text/plain", "full_name=ok", false},
		{"years=20", "", "full_name=ok", false},
		{"years=20", "application/json", `{"name":"ok","extra":1}`, true},
		{"years=20&extra=1", "application/json", `{"name":"ok"}`, true},
		{"years=20", "application/x-www-form-urlencoded", "full_name=ok&extra=1", true},
		{"years=20", "application/json", `{"name":" "}`, false},
	} {
		original := requestUser{Name: "old", Age: 30, Tags: []string{"old"}}
		out := original
		err := shape.BindRequest(&out, parameterRequest(tc.query, tc.media, tc.body), shape.RequestOptions{Precedence: shape.QueryFirst, DisallowUnknownFields: tc.strict})
		if err == nil || !reflect.DeepEqual(out, original) {
			t.Fatalf("query=%s body=%s out=%#v err=%v", tc.query, tc.body, out, err)
		}
	}
}

func TestBindRequestMultipart(t *testing.T) {
	for _, file := range []bool{false, true} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for _, field := range [][2]string{{"full_name", " Pong "}, {"tag", "a"}, {"tag", "b"}} {
			if err := writer.WriteField(field[0], field[1]); err != nil {
				t.Fatal(err)
			}
		}
		if file {
			part, err := writer.CreateFormFile("file", "test.txt")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(part, "content"); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		r := parameterRequest("years=20", writer.FormDataContentType(), body.String())
		var out requestUser
		err := shape.BindRequest(&out, r)
		if file {
			if !errors.Is(err, shape.ErrRequestFiles) || !reflect.DeepEqual(out, requestUser{}) {
				t.Fatal(err)
			}
		} else if err != nil || out.Name != "Pong" || out.Age != 20 || !reflect.DeepEqual(out.Tags, []string{"a", "b"}) {
			t.Fatalf("out=%#v err=%v", out, err)
		}
		if r.MultipartForm != nil {
			t.Fatal("BindRequest created multipart storage")
		}
	}
	for _, media := range []string{"multipart/form-data", "multipart/form-data; boundary=wrong"} {
		var out requestUser
		if err := shape.BindRequest(&out, parameterRequest("years=20", media, "broken")); err == nil {
			t.Fatal("accepted malformed multipart")
		}
	}
}

func TestBindRequestLimitsAndCancellation(t *testing.T) {
	type Request struct {
		Name string `json:"name"`
	}
	body := `{"name":"ok"}`
	var out Request
	if err := shape.BindRequest(&out, parameterRequest("", "application/json", body), shape.RequestOptions{MaxBytes: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body string
		max  int64
	}{{body, int64(len(body) - 1)}, {strings.Repeat(" ", int(shape.DefaultMaxRequestBytes)+1), 0}} {
		original := out
		err := shape.BindRequest(&out, parameterRequest("", "application/json", tc.body), shape.RequestOptions{MaxBytes: tc.max})
		if !errors.Is(err, shape.ErrRequestTooLarge) || out != original {
			t.Fatalf("out=%#v err=%v", out, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := shape.BindRequest(&out, parameterRequest("", "application/json", body).WithContext(ctx)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := shape.BindRequest(&out, nil); err == nil {
		t.Fatal("nil request accepted")
	}
	if err := shape.BindRequest[Request](nil, parameterRequest("", "", "")); err == nil {
		t.Fatal("nil target accepted")
	}
	if err := shape.BindRequest(&out, parameterRequest("", "text/plain", "x")); !errors.Is(err, shape.ErrUnsupportedContentType) {
		t.Fatal(err)
	}
	r := parameterRequest("", "application/x-www-form-urlencoded", "Name=ok")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	if err := shape.BindRequest(&out, r); err == nil {
		t.Fatal("already consumed form was accepted")
	}
}

type requestRootCodec struct{ Name string }

func (*requestRootCodec) UnmarshalJSON([]byte) error { panic("root codec must not run") }

type cancelRequestBody struct{ cancel context.CancelFunc }

func (r cancelRequestBody) Read([]byte) (int, error) { r.cancel(); return 0, io.EOF }
func (cancelRequestBody) Close() error               { return nil }

func TestBindRequestEdgeCases(t *testing.T) {
	type Request struct {
		Name string `shape:"ifzero=guest"`
	}
	for _, r := range []*http.Request{
		{URL: &url.URL{}},
		parameterRequest("", "application/json", ""),
	} {
		out := Request{Name: "old"}
		if err := shape.BindRequest(&out, r); err != nil || out.Name != "guest" {
			t.Fatalf("out=%#v err=%v", out, err)
		}
	}
	var root requestRootCodec
	if err := shape.BindRequest(&root, parameterRequest("", "application/json", `{"Name":"x"}`)); err == nil {
		t.Fatal("root codec accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := parameterRequest("", "application/json", "").WithContext(ctx)
	r.Body = cancelRequestBody{cancel}
	out := Request{Name: "old"}
	if err := shape.BindRequest(&out, r); !errors.Is(err, context.Canceled) || out.Name != "old" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	for _, options := range [][]shape.RequestOptions{
		{{}, {}}, {{MaxBytes: -1}}, {{Precedence: shape.RequestPrecedence(255)}},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid configuration accepted")
				}
			}()
			_ = shape.BindRequest(&out, parameterRequest("", "", ""), options...)
		}()
	}
}

func TestBindRequestJSONExactNamePrecedesFoldedName(t *testing.T) {
	type Request struct {
		First  string `json:"name" query:"first"`
		Second string `json:"NAME" query:"second"`
	}
	var out Request
	err := shape.BindRequest(&out, parameterRequest("first=query", "application/json", `{"NAME":"body"}`))
	if err != nil || out.First != "query" || out.Second != "body" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	var conflict *shape.SourceConflictError
	if err := shape.BindRequest(&out, parameterRequest("second=query", "application/json", `{"NAME":"body"}`)); !errors.As(err, &conflict) || conflict.Field != "Second" {
		t.Fatal(err)
	}
}

func TestBindRequestChecksAllInputPlans(t *testing.T) {
	type Invalid struct {
		Values map[string]int `json:"values"`
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("unsupported parameter type was accepted for JSON-only input")
			}
		}()
		var out Invalid
		_ = shape.BindRequest(&out, parameterRequest("", "application/json", `{"values":{"x":1}}`))
	}()
	type Valid struct {
		Values map[string]int `json:"values" query:"-" form:"-"`
	}
	var out Valid
	err := shape.BindRequest(&out, parameterRequest("", "application/json", `{"values":{"x":1}}`))
	if err != nil || out.Values["x"] != 1 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestBindRequestProtectsCustomDecoderStorage(t *testing.T) {
	type Request struct {
		Items sharedItems `json:"items" query:"-" form:"-"`
		Age   int         `query:"age" shape:"min=18"`
	}
	sharedDecodedItems = sharedItems{{Name: " original "}}
	out := Request{Items: sharedDecodedItems, Age: 30}
	err := shape.BindRequest(&out, parameterRequest("age=1", "application/json", `{"items":[]}`))
	if err == nil || out.Items[0].Name != " original " || out.Age != 30 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestBindRequestConcurrentSources(t *testing.T) {
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				var out requestUser
				if err := shape.BindRequest(&out, parameterRequest("years=20", "application/json", `{"name":" Pong "}`)); err != nil || out.Name != "Pong" {
					t.Errorf("out=%#v err=%v", out, err)
				}
			}
		}()
	}
	wg.Wait()
}

func FuzzBindRequest(f *testing.F) {
	f.Add("years=20", `{"name":"Pong"}`, false)
	f.Add("q=Pong&years=20", "full_name=other", true)
	f.Add("q=%GG", "bad", false)
	f.Fuzz(func(t *testing.T, query, body string, form bool) {
		media := "application/json"
		if form {
			media = "application/x-www-form-urlencoded"
		}
		r := &http.Request{URL: &url.URL{RawQuery: query}, Header: http.Header{"Content-Type": {media}}, Body: io.NopCloser(strings.NewReader(body))}
		original := requestUser{Name: "old", Age: 30}
		out := original
		err := shape.BindRequest(&out, r, shape.RequestOptions{MaxBytes: 4096, DisallowUnknownFields: true})
		if err != nil {
			if !reflect.DeepEqual(out, original) {
				t.Fatal("failed bind mutated target")
			}
			return
		}
		if err := shape.FromTags[requestUser]().Validate(out); err != nil {
			t.Fatal(err)
		}
	})
}
