package shape_test

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
	"github.com/rhevorn/shape/types"
	"github.com/rhevorn/shape/validate"
)

type parameterAddress struct {
	City string `json:"city" form:"town" query:"place" shape:"trim,notempty"`
}

type parameterDoc struct {
	Name    string            `json:"name" form:"full_name" query:"q" shape:"trim,notempty"`
	Age     int               `json:"age" shape:"min=18" form:"age" query:"age"`
	Tags    []string          `json:"tags" form:"tags" query:"tags"`
	Address *parameterAddress `json:"address" form:"home" query:"where"`
	Nick    *string           `json:"nick" form:"nick" query:"nick"`
	Enabled bool              `json:"enabled" form:"enabled" query:"enabled"`
	Timeout types.Duration    `json:"timeout" form:"timeout" query:"timeout"`
	Created time.Time         `json:"created" form:"created" query:"created"`
}

func explicitParameterSchema() shape.StructSpec[parameterDoc] {
	return shape.New[parameterDoc](
		shape.Field("Name", shape.String().Trim().NotEmpty()),
		shape.Field("Age", shape.Int().Min(18)),
		shape.Field("Address", shape.Pointer(shape.New[parameterAddress](shape.Field("City", shape.String().Trim().NotEmpty())))),
	)
}

func TestParameterEntrypoints(t *testing.T) {
	for _, source := range []string{"form", "query"} {
		t.Run(source, func(t *testing.T) {
			name, city := "full_name", "home.town"
			if source == "query" {
				name, city = "q", "where.place"
			}
			values := url.Values{name: {" Pong "}, "age": {"20"}, "tags": {"go,web", "shape"}, city: {" Shanghai "}, "nick": {""}, "enabled": {"1"}, "timeout": {"30s"}, "created": {"2026-01-02T03:04:05Z"}}
			tagged, explicit := shape.FromTags[parameterDoc](), explicitParameterSchema()
			var parsers []func(url.Values) (parameterDoc, error)
			if source == "form" {
				parsers = append(parsers,
					func(v url.Values) (parameterDoc, error) { return tagged.ParseForm(v) },
					func(v url.Values) (parameterDoc, error) { return explicit.ParseForm(v) },
					func(v url.Values) (parameterDoc, error) { return shape.ParseForm(&tagged, v) },
					func(v url.Values) (parameterDoc, error) {
						var out parameterDoc
						err := shape.BindForm(&out, v)
						return out, err
					},
				)
			} else {
				parsers = append(parsers,
					func(v url.Values) (parameterDoc, error) { return tagged.ParseQuery(v) },
					func(v url.Values) (parameterDoc, error) { return explicit.ParseQuery(v) },
					func(v url.Values) (parameterDoc, error) { return shape.ParseQuery(&explicit, v) },
					func(v url.Values) (parameterDoc, error) {
						var out parameterDoc
						err := shape.BindQuery(&out, v)
						return out, err
					},
				)
			}
			for _, parse := range parsers {
				out, err := parse(values)
				if err != nil {
					t.Fatal(err)
				}
				if out.Name != "Pong" || out.Age != 20 || out.Address == nil || out.Address.City != "Shanghai" || out.Nick == nil || *out.Nick != "" || !out.Enabled || out.Timeout != types.Duration(30*time.Second) || out.Created.Format(time.RFC3339) != "2026-01-02T03:04:05Z" || !reflect.DeepEqual(out.Tags, []string{"go,web", "shape"}) {
					t.Fatalf("unexpected result: %#v", out)
				}
				out.Tags[0] = "changed"
				if values["tags"][0] != "go,web" || values[name][0] != " Pong " {
					t.Fatal("input was mutated")
				}
			}
		})
	}
}

func TestParameterDecodingErrors(t *testing.T) {
	type Request struct {
		Number   int8    `json:"number" form:"number" query:"number"`
		Unsigned uint8   `json:"unsigned" form:"unsigned" query:"unsigned"`
		Float    float32 `json:"float" form:"float" query:"float"`
		Bool     bool    `json:"bool" form:"bool" query:"bool"`
		List     []int   `json:"list" form:"list" query:"list"`
		Child    struct {
			Name string `json:"name" form:"name" query:"name"`
		} `json:"child" form:"child" query:"child"`
	}
	schema := shape.FromTags[Request]()
	cases := []struct {
		key    string
		values []string
		path   string
	}{
		{"number", []string{""}, "number"}, {"number", []string{"1", "2"}, "number"},
		{"number", nil, "number"}, {"number", []string{}, "number"},
		{"number", []string{"128"}, "number"}, {"number", []string{"0x10"}, "number"},
		{"number", []string{" 1"}, "number"}, {"unsigned", []string{"-1"}, "unsigned"},
		{"unsigned", []string{"256"}, "unsigned"}, {"float", []string{"1e100"}, "float"},
		{"float", []string{"NaN"}, "float"}, {"float", []string{"Inf"}, "float"},
		{"float", []string{"0x1p2"}, "float"}, {"float", []string{"1_000"}, "float"},
		{"bool", []string{"on"}, "bool"}, {"bool", []string{"TRUE"}, "bool"},
		{"list", []string{"1", "bad"}, "list[1]"}, {"child", []string{"bad"}, "child"},
	}
	for _, tc := range cases {
		for _, source := range []string{"form", "query"} {
			var out Request
			var err error
			if source == "form" {
				out, err = schema.ParseForm(url.Values{tc.key: tc.values})
			} else {
				out, err = schema.ParseQuery(url.Values{tc.key: tc.values})
			}
			var decode *shape.ParameterError
			if !errors.As(err, &decode) || decode.Path.String() != tc.path || decode.Source != source || !reflect.DeepEqual(out, Request{}) {
				t.Errorf("%s %s=%v: out=%#v err=%v", source, tc.key, tc.values, out, err)
			}
		}
	}
	_, err := schema.ParseQuery(url.Values{"number": {"128"}})
	var number *strconv.NumError
	if !errors.As(err, &number) || !errors.Is(err, strconv.ErrRange) {
		t.Fatalf("lost cause: %v", err)
	}
	for _, value := range []string{"true", "false", "1", "0"} {
		if _, err := schema.ParseForm(url.Values{"bool": {value}}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParameterUnknownFieldsAndCachedErrorPaths(t *testing.T) {
	type Request struct {
		Number int `json:"number" form:"number" query:"number"`
	}
	schema := shape.FromTags[Request]()
	values := url.Values{"z": {"1"}, "a": {"2"}}
	if _, err := schema.ParseForm(values); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		_, err := schema.ParseQuery(values, shape.QueryOptions{DisallowUnknownFields: true})
		var decode *shape.ParameterError
		if !errors.As(err, &decode) || decode.Path.String() != "a" {
			t.Fatal(err)
		}
	}
	for range 2 {
		_, err := schema.ParseForm(url.Values{"number": {"bad"}}, shape.FormOptions{DisallowUnknownFields: true})
		var decode *shape.ParameterError
		if !errors.As(err, &decode) || decode.Path.String() != "number" {
			t.Fatal(err)
		}
		decode.Path[0].Key = "changed"
	}
}

func TestParameterRulesAreIndependentOfSource(t *testing.T) {
	type Request struct {
		Name   string `json:"name" form:"f" query:"q" shape:"notempty"`
		Hidden int    `json:"-" form:"-" query:"-" shape:"ifzero=18,min=18"`
	}
	tagged := shape.FromTags[Request]()
	explicit := shape.New[Request](shape.Field("Name", shape.String().NotEmpty()), shape.Field("Hidden", shape.Int().IfZero(18).Min(18)))
	for _, schema := range []shape.Schema[Request]{tagged, explicit} {
		parsers := []func() (Request, error){
			func() (Request, error) { return shape.ParseJSON(schema, []byte(`{"name":"ok"}`)) },
			func() (Request, error) { return shape.ParseForm(schema, url.Values{"f": {"ok"}}) },
			func() (Request, error) { return shape.ParseQuery(schema, url.Values{"q": {"ok"}}) },
		}
		want, err := schema.Transform(Request{Name: "ok"})
		if err != nil {
			t.Fatal(err)
		}
		for _, parse := range parsers {
			out, err := parse()
			if err != nil || out != want || out.Hidden != 18 {
				t.Fatalf("out=%#v want=%#v err=%v", out, want, err)
			}
			if err := schema.Validate(out); err != nil {
				t.Fatalf("parsed value failed validation: %v", err)
			}
		}
	}
	type Required struct {
		Token string `json:"-" form:"-" query:"-" shape:"notempty"`
	}
	for _, schema := range []shape.Schema[Required]{shape.FromTags[Required](), shape.New[Required](shape.Field("Token", shape.String().NotEmpty()))} {
		for _, parse := range []func() (Required, error){
			func() (Required, error) { return shape.ParseJSON(schema, []byte(`{}`)) },
			func() (Required, error) { return shape.ParseForm(schema, nil) },
			func() (Required, error) { return shape.ParseQuery(schema, nil) },
		} {
			_, err := parse()
			var validation *validate.Error
			if !errors.As(err, &validation) || validation.Issues[0].Path.String() != "Token" {
				t.Fatalf("excluded required field was skipped: %v", err)
			}
		}
	}
}

func TestParameterTagsAreIndependent(t *testing.T) {
	type Request struct {
		Name   string `json:"json_name" form:"form_name" query:"query_name"`
		Hidden string `json:"-"`
	}
	schema := shape.FromTags[Request]()
	form, err := schema.ParseForm(url.Values{"form_name": {"f"}, "Hidden": {"visible"}}, shape.FormOptions{DisallowUnknownFields: true})
	if err != nil || form.Name != "f" || form.Hidden != "visible" {
		t.Fatalf("form=%#v err=%v", form, err)
	}
	query, err := schema.ParseQuery(url.Values{"query_name": {"q"}, "Hidden": {"visible"}}, shape.QueryOptions{DisallowUnknownFields: true})
	if err != nil || query.Name != "q" || query.Hidden != "visible" {
		t.Fatalf("query=%#v err=%v", query, err)
	}
	for _, key := range []string{"json_name", "query_name", "Name"} {
		if _, err := schema.ParseForm(url.Values{key: {"bad"}}, shape.FormOptions{DisallowUnknownFields: true}); err == nil {
			t.Fatalf("form accepted %s", key)
		}
	}
	for _, key := range []string{"json_name", "form_name", "Name"} {
		if _, err := schema.ParseQuery(url.Values{key: {"bad"}}, shape.QueryOptions{DisallowUnknownFields: true}); err == nil {
			t.Fatalf("query accepted %s", key)
		}
	}
	type Fallback struct {
		Name  string `json:"name"`
		Empty string `json:"empty" form:"" query:""`
	}
	fallback := shape.FromTags[Fallback]()
	out, err := fallback.ParseForm(url.Values{"Name": {"a"}, "Empty": {"b"}}, shape.FormOptions{DisallowUnknownFields: true})
	if err != nil || out.Name != "a" || out.Empty != "b" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	if _, err := fallback.ParseQuery(url.Values{"name": {"x"}}, shape.QueryOptions{DisallowUnknownFields: true}); err == nil {
		t.Fatal("query fell back to json tag")
	}
}

func TestParameterMissingDefaultsAndNestedPaths(t *testing.T) {
	type NamedPointer *parameterAddress
	type Request struct {
		Name    string       `json:"name" shape:"ifzero=guest" form:"name" query:"name"`
		Nick    *string      `json:"nick" shape:"ifnull=default" form:"nick" query:"nick"`
		Address NamedPointer `json:"address" form:"address" query:"address"`
	}
	schema := shape.FromTags[Request]()
	out, err := schema.ParseForm(nil)
	if err != nil || out.Name != "guest" || out.Nick == nil || *out.Nick != "default" || out.Address != nil {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	out, err = schema.ParseForm(url.Values{"nick": {""}, "address.town": {"Shanghai"}})
	if err != nil || *out.Nick != "" || out.Address.City != "Shanghai" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	_, err = schema.ParseQuery(url.Values{"address.place": {" "}})
	var validation *validate.Error
	if !errors.As(err, &validation) || validation.Issues[0].Path.String() != "address.place" {
		t.Fatalf("path=%v", err)
	}
	failure := errors.New("transform failure")
	nested := shape.New[parameterAddress](shape.Field("City", shape.String().Apply(func(string) (string, error) { return "", failure })))
	explicit := shape.New[struct {
		Address parameterAddress `form:"home"`
	}](shape.Field("Address", nested))
	_, err = explicit.ParseForm(url.Values{"home.town": {"x"}})
	var transform *shape.TransformError
	if !errors.As(err, &transform) || transform.Path.String() != "home.town" || !errors.Is(err, failure) {
		t.Fatalf("path=%v", err)
	}
}

func TestParameterBindIsAtomicAndReplacesTarget(t *testing.T) {
	type Request struct {
		Name string   `json:"name" shape:"trim,notempty" form:"name" query:"name"`
		Age  int      `json:"age" shape:"min=18" form:"age" query:"age"`
		Tags []string `json:"tags" form:"tags" query:"tags"`
	}
	for _, bind := range []func(context.Context, *Request, url.Values) error{
		func(ctx context.Context, target *Request, values url.Values) error {
			return shape.BindFormContext(ctx, target, values)
		},
		func(ctx context.Context, target *Request, values url.Values) error {
			return shape.BindQueryContext(ctx, target, values)
		},
	} {
		original := Request{Name: "old", Age: 40, Tags: []string{"old"}}
		for _, values := range []url.Values{{"name": {"new"}, "age": {"bad"}}, {"name": {"new"}, "age": {"0"}}, {"name": {" "}, "age": {"20"}}} {
			target := original
			if err := bind(context.Background(), &target, values); err == nil || !reflect.DeepEqual(target, original) {
				t.Fatalf("target=%#v err=%v", target, err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		target := original
		if err := bind(ctx, &target, url.Values{"name": {"new"}, "age": {"20"}}); !errors.Is(err, context.Canceled) || !reflect.DeepEqual(target, original) {
			t.Fatalf("target=%#v err=%v", target, err)
		}
		if err := bind(context.Background(), &target, url.Values{"name": {"new"}, "age": {"20"}}); err != nil || target.Name != "new" || target.Tags != nil {
			t.Fatalf("replacement=%#v err=%v", target, err)
		}
		if err := bind(context.Background(), nil, nil); err == nil {
			t.Fatal("nil target accepted")
		}
	}
}

func TestParameterCustomDecoderOwnershipAndOverrides(t *testing.T) {
	type Doc struct {
		Value sharedTextValue `json:"value" form:"value" query:"value"`
	}
	sharedTextItems = []int{1}
	change := func(v Doc) (Doc, error) { v.Value.Items[0] = 2; return v, errors.New("failed") }
	for _, schema := range []shape.Schema[Doc]{shape.FromTags[Doc]().Apply(change), shape.New[Doc]().Apply(change)} {
		if _, err := shape.ParseForm(schema, url.Values{"value": {"cached"}}); err == nil {
			t.Fatal("missing failure")
		}
		if sharedTextItems[0] != 1 {
			t.Fatal("custom decoder storage was mutated")
		}
	}
	override := overriddenTransformSchema{shape.New[embeddedDoc]()}
	if out, err := shape.ParseQuery[embeddedDoc](override, url.Values{"name": {"original"}}); err != nil || out.Name != "overridden" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schema := shape.FromTags[embeddedDoc]().RefineContext(func(context.Context, embeddedDoc) error { cancel(); return nil })
	if out, err := schema.ParseQueryContext(ctx, url.Values{"name": {"x"}}); !errors.Is(err, context.Canceled) || out.Name != "" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestParameterConcurrentSources(t *testing.T) {
	schema := shape.FromTags[parameterAddress]()
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				for _, run := range []func() (parameterAddress, error){
					func() (parameterAddress, error) { return schema.ParseJSON([]byte(`{"city":"ok"}`)) },
					func() (parameterAddress, error) { return schema.ParseForm(url.Values{"town": {"ok"}}) },
					func() (parameterAddress, error) { return schema.ParseQuery(url.Values{"place": {"ok"}}) },
				} {
					if out, err := run(); err != nil || out.City != "ok" {
						t.Errorf("out=%#v err=%v", out, err)
					}
				}
			}
		}()
	}
	wg.Wait()
}

type parameterText string

func (v *parameterText) UnmarshalText(text []byte) error {
	if string(text) == "bad" {
		return strconv.ErrSyntax
	}
	*v = parameterText("decoded:" + string(text))
	return nil
}

func TestParameterTextTypesAndScalarPointers(t *testing.T) {
	type NamedInt int16
	type NamedPointer *NamedInt
	type Request struct {
		Text    parameterText   `json:"text" form:"text" query:"text"`
		Texts   []parameterText `json:"texts" form:"texts" query:"texts"`
		Ints    []NamedPointer  `json:"ints" form:"ints" query:"ints"`
		Bytes   []byte          `json:"bytes" form:"bytes" query:"bytes"`
		Timeout *types.Duration `json:"timeout" form:"timeout" query:"timeout"`
	}
	schema := shape.FromTags[Request]()
	out, err := schema.ParseForm(url.Values{"text": {"a"}, "texts": {"b", "c"}, "ints": {"1", "2"}, "bytes": {"0", "255"}, "timeout": {"-500ms"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "decoded:a" || !reflect.DeepEqual(out.Texts, []parameterText{"decoded:b", "decoded:c"}) || len(out.Ints) != 2 || *out.Ints[0] != 1 || *out.Ints[1] != 2 || !reflect.DeepEqual(out.Bytes, []byte{0, 255}) || *out.Timeout != types.Duration(-500*time.Millisecond) {
		t.Fatalf("out=%#v", out)
	}
	for _, key := range []string{"text", "texts"} {
		_, err := schema.ParseQuery(url.Values{key: {"bad"}})
		if !errors.Is(err, strconv.ErrSyntax) {
			t.Fatalf("lost text-decoder error: %v", err)
		}
	}
	if _, err := schema.ParseForm(url.Values{"timeout": {""}}); err == nil {
		t.Fatal("accepted an empty duration")
	}
	// Parameter duration syntax must not change the existing JSON integer map keys.
	value, err := shape.ParseJSON(shape.Map(shape.Duration(), shape.String()), []byte(`{"1000":"ok"}`))
	if err != nil || value[types.Duration(1000)] != "ok" {
		t.Fatalf("JSON map=%v err=%v", value, err)
	}
}

func TestParameterEmptyTagsAndExcludedGraphs(t *testing.T) {
	type Request struct {
		Fallback string         `json:"fallback" form:"" query:""`
		Explicit string         `json:"-" form:"" query:""`
		Map      map[string]int `json:"map" form:"-" query:"-"`
	}
	schema := shape.FromTags[Request]()
	values := url.Values{"Fallback": {"a"}, "Explicit": {"b"}}
	for _, run := range []func() (Request, error){
		func() (Request, error) {
			return schema.ParseForm(values, shape.FormOptions{DisallowUnknownFields: true})
		},
		func() (Request, error) {
			return schema.ParseQuery(values, shape.QueryOptions{DisallowUnknownFields: true})
		},
	} {
		out, err := run()
		if err != nil || out.Fallback != "a" || out.Explicit != "b" || out.Map != nil {
			t.Fatalf("out=%#v err=%v", out, err)
		}
	}
}

func TestJSONExportRejectsNonJSONRules(t *testing.T) {
	type Child struct {
		Secret string `json:"-" form:"secret" shape:"notempty"`
	}
	type Request struct {
		Child Child `json:"child"`
	}
	schema := shape.FromTags[Request]()
	if _, err := schema.ParseJSON([]byte(`{"child":null}`)); err == nil {
		t.Fatal("JSON skipped the required field")
	}
	var unsupported *shape.UnsupportedSchemaError
	if _, err := jsonschema.Export(schema); !errors.As(err, &unsupported) {
		t.Fatalf("export silently omitted a rule: %v", err)
	}
	type Harmless struct {
		Hidden int `json:"-" shape:"min=0"`
	}
	if _, err := jsonschema.Export(shape.FromTags[Harmless]()); err != nil {
		t.Fatalf("zero-compatible hidden field: %v", err)
	}
}

type parameterCancelSchema struct {
	shape.TaggedSpec[embeddedDoc]
	cancel context.CancelFunc
}

func (s parameterCancelSchema) ValidateContext(context.Context, embeddedDoc) error {
	s.cancel()
	return errors.New("interrupted validation")
}

func TestParameterCancellationTakesPrecedenceOverCallbackError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schema := parameterCancelSchema{shape.FromTags[embeddedDoc](), cancel}
	out, err := shape.ParseFormContext[embeddedDoc](ctx, schema, url.Values{"name": {"x"}})
	if !errors.Is(err, context.Canceled) || out.Name != "" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestParameterInvalidDefinitions(t *testing.T) {
	type ConflictingPaths struct {
		Name  string `json:"Other" form:"Other" query:"Other"`
		Other string `json:"-" form:"other"`
	}
	panics := []func(){
		func() { _ = shape.FromTags[ConflictingPaths]() },
		func() {
			_ = shape.New[ConflictingPaths](shape.Field("Name", shape.String()), shape.Field("Other", shape.String()))
		},
		func() { _, _ = shape.ParseForm(shape.Value[map[string]string](), nil) },
		func() { _, _ = shape.ParseForm(shape.New[struct{ Value map[string]string }](), nil) },
		func() { _, _ = shape.ParseQuery(shape.New[struct{ Value []struct{ Name string } }](), nil) },
		func() { _, _ = shape.ParseForm(shape.New[struct{ Value [2]int }](), nil) },
		func() { _, _ = shape.ParseQuery(shape.New[struct{ Value any }](), nil) },
		func() { _, _ = shape.ParseForm(shape.New[struct{ Value **int }](), nil) },
		func() {
			_, _ = shape.ParseForm(shape.New[struct {
				Value string `form:"a.b"`
			}](), nil)
		},
		func() {
			_, _ = shape.ParseQuery(shape.New[struct {
				A string `query:"x"`
				B string `query:"x"`
			}](), nil)
		},
		func() { _, _ = shape.FromTags[embeddedDoc]().ParseFormContext(nil, nil) },
		func() { _, _ = shape.FromTags[embeddedDoc]().ParseForm(nil, shape.FormOptions{}, shape.FormOptions{}) },
		func() {
			_, _ = shape.FromTags[embeddedDoc]().ParseQuery(nil, shape.QueryOptions{}, shape.QueryOptions{})
		},
	}
	for i, call := range panics {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("expected configuration panic")
				}
			}()
			call()
		})
	}
}

func FuzzParameterDecoding(f *testing.F) {
	for _, seed := range []string{"name=Pong&age=20&tags=a&tags=b", "age=999999999999999999999", "name=%FF", "child.name=x", "age=1&age=2", "name=%GG", "name=x;y"} {
		f.Add(seed)
	}
	type Request struct {
		Name  string   `json:"name" shape:"trim,notempty" form:"name" query:"name"`
		Age   int16    `json:"age" shape:"min=18" form:"age" query:"age"`
		Tags  []string `json:"tags" form:"tags" query:"tags"`
		Child *struct {
			Name string `json:"name" form:"name" query:"name"`
		} `json:"child" form:"child" query:"child"`
	}
	schema := shape.FromTags[Request]()
	f.Fuzz(func(t *testing.T, raw string) {
		values, err := url.ParseQuery(raw)
		if err != nil {
			return
		}
		a, ae := schema.ParseForm(values, shape.FormOptions{DisallowUnknownFields: true})
		b, be := schema.ParseQuery(values, shape.QueryOptions{DisallowUnknownFields: true})
		if (ae == nil) != (be == nil) || !reflect.DeepEqual(a, b) {
			t.Fatalf("form/query differ: %#v %v / %#v %v", a, ae, b, be)
		}
		if ae == nil {
			if err := schema.Validate(a); err != nil {
				t.Fatalf("accepted invalid value: %v", err)
			}
		} else if !reflect.DeepEqual(a, Request{}) {
			t.Fatal("error returned a partial value")
		}
	})
}
