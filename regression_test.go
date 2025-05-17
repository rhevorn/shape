package shape_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

func TestErrorMustNotMutateSchema(t *testing.T) {
	s := shape.String().OneOf("admin", "user")
	var ve *validate.Error
	errors.As(s.Validate("bad"), &ve)
	ve.Issues[0].Expected.([]string)[0] = "attacker"
	if s.Validate("attacker") == nil {
		t.Fatal("editing an error changed the schema's accepted values")
	}
}

func TestAndPreservesDeclarationOrder(t *testing.T) {
	var got []string
	rule := func(name string) func(string) error {
		return func(string) error { got = append(got, name); return nil }
	}
	s := validate.String().Refine(rule("a")).And(validate.String().Refine(rule("b"))).Refine(rule("c"))
	_ = s.Validate("x")
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("got %v", got)
	}
}

func TestExplicitCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := shape.New[struct{}]()
	if err := s.ValidateContext(ctx, struct{}{}); !errors.Is(err, context.Canceled) {
		t.Errorf("ValidateContext = %v", err)
	}
	if _, err := s.TransformContext(ctx, struct{}{}); !errors.Is(err, context.Canceled) {
		t.Errorf("TransformContext = %v", err)
	}
}

func nonnull(d map[string]any) map[string]any {
	if branches, ok := d["anyOf"].([]any); ok {
		return branches[0].(map[string]any)
	}
	return d
}

func TestExportKeepsAllConstraints(t *testing.T) {
	type DTO struct {
		Name string `json:"name" shape:"minlength=5,notempty"`
	}
	s := shape.FromTags[DTO]()
	d, err := jsonschema.Export(s)
	if err != nil {
		t.Fatal(err)
	}
	p := nonnull(d)["properties"].(map[string]any)["name"].(map[string]any)
	if s.Validate(DTO{Name: "x"}) == nil {
		t.Fatal("bad fixture")
	}
	if p["minLength"] != 5 {
		t.Fatalf("runtime requires 5 but export = %v", p)
	}
}

func TestExportJSONRepresentation(t *testing.T) {
	type DTO struct {
		N    int    `json:"n,string" shape:"min=1"`
		Data []byte `json:"data" shape:"notempty"`
	}
	s := shape.FromTags[DTO]()
	out, err := s.ParseJSON([]byte(`{"n":"2","data":"AQI="}`))
	if err != nil {
		t.Fatal(err)
	}
	d, err := jsonschema.Export(s)
	if err != nil {
		var unsupported *shape.UnsupportedSchemaError
		if !errors.As(err, &unsupported) {
			t.Fatal(err)
		}
		return
	}
	props := nonnull(d)["properties"].(map[string]any)
	raw, _ := json.Marshal(out)
	if props["n"].(map[string]any)["type"] != "string" || props["data"].(map[string]any)["type"] != "string" {
		t.Fatalf("valid wire JSON = %s; exported properties = %v", raw, props)
	}
}

func TestExportDocumentIsDetached(t *testing.T) {
	type DTO struct {
		Role string `json:"role" shape:"oneof=admin|user"`
	}
	s := shape.FromTags[DTO]()
	first, _ := jsonschema.Export(s)
	first["properties"].(map[string]any)["role"].(map[string]any)["enum"].([]any)[0] = "attacker"
	second, _ := jsonschema.Export(s)
	got := second["properties"].(map[string]any)["role"].(map[string]any)["enum"].([]any)[0]
	if got != "admin" {
		t.Fatalf("editing first document poisoned subsequent exports: %v", got)
	}
}

func TestPointerFallbackMatchesTags(t *testing.T) {
	type DTO struct {
		Name *string `json:"name" shape:"ifnull=' guest ',trim"`
	}
	fallback := " guest "
	explicit := shape.New[DTO](shape.Field("Name", shape.Pointer(shape.String().Trim()).IfNull(&fallback)))
	a, err := explicit.ParseJSON([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := shape.FromTags[DTO]().ParseJSON([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if *a.Name != *b.Name {
		t.Fatalf("explicit=%q, tagged=%q", *a.Name, *b.Name)
	}
}

func TestNamedStringMapPath(t *testing.T) {
	type Key string
	s := shape.Map(shape.Value[Key](), shape.Int().Positive())
	var ve *validate.Error
	errors.As(s.Validate(map[Key]int{"3": 0}), &ve)
	if got := ve.Issues[0].Path.String(); got != `["3"]` {
		t.Fatalf("named string key rendered as %s", got)
	}
}

func TestComposedTransformPath(t *testing.T) {
	inner := transform.Int().Apply(func(int) (int, error) { return 0, errors.New("bad") })
	composed := transform.Slice(transform.Slice(inner))
	s := shape.Value[[][]int]().Apply(composed.Transform)
	_, err := s.Transform([][]int{{1}})
	var te *shape.TransformError
	if !errors.As(err, &te) {
		t.Fatalf("expected TransformError: %v", err)
	}
	if got := te.Path.String(); got != "[0][0]" {
		t.Fatalf("two-dimensional path became %s", got)
	}
}

func TestEnumErrorsAreIndependentDuringConcurrentReuse(t *testing.T) {
	type Role int
	number := shape.Number[Role]().OneOf(1, 2)
	text := shape.String().OneOf("a", "b")
	type DTO struct {
		Role string `shape:"oneof=a|b"`
	}
	tagged := shape.FromTags[DTO]()
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				var ve *validate.Error
				errors.As(number.Validate(3), &ve)
				ve.Issues[0].Expected.([]Role)[0] = 3
				errors.As(text.Validate("c"), &ve)
				ve.Issues[0].Expected.([]string)[0] = "c"
				errors.As(tagged.Validate(DTO{Role: "c"}), &ve)
				ve.Issues[0].Expected.([]any)[0] = "c"
			}
		}()
	}
	wg.Wait()
	if number.Validate(1) != nil || number.Validate(3) == nil || text.Validate("a") != nil || text.Validate("c") == nil {
		t.Fatal("enum changed after errors were edited")
	}
	var ve *validate.Error
	errors.As(tagged.Validate(DTO{Role: "c"}), &ve)
	if ve.Issues[0].Expected.([]any)[0] != "a" {
		t.Fatal("tagged expected values escaped")
	}
}

func TestTaggedPruningRetainsFiniteChecksAndIndependentPaths(t *testing.T) {
	type Child struct {
		Value float64 `json:"value"`
	}
	type DTO struct {
		Children []Child `json:"children"`
	}
	s := shape.FromTags[DTO]()
	var ve *validate.Error
	if !errors.As(s.Validate(DTO{Children: []Child{{math.NaN()}, {math.Inf(1)}}}), &ve) {
		t.Fatal("non-finite elements passed")
	}
	if len(ve.Issues) != 2 || ve.Issues[0].Path.String() != "children[0].value" || ve.Issues[1].Path.String() != "children[1].value" {
		t.Fatalf("paths = %#v", ve)
	}
}

func TestContainerFallbacksRunThroughInnerTransforms(t *testing.T) {
	inner := shape.String().Trim().ToLower()
	slice := shape.Slice(inner).IfNull([]string{" A "})
	out, err := slice.Transform(nil)
	if err != nil || !reflect.DeepEqual(out, []string{"a"}) {
		t.Fatalf("slice=%v, %v", out, err)
	}
	mapping := shape.Map(inner, inner).IfNull(map[string]string{" A ": " B "})
	got, err := mapping.Transform(nil)
	if err != nil || !reflect.DeepEqual(got, map[string]string{"a": "b"}) {
		t.Fatalf("map=%v, %v", got, err)
	}
	added := shape.Slice(inner).Apply(func(v []string) ([]string, error) { return append(v, " C "), nil })
	out, err = added.Transform([]string{" A "})
	if err != nil || !reflect.DeepEqual(out, []string{"a", "c"}) {
		t.Fatalf("added=%v, %v", out, err)
	}
}

func TestAndValidateFirstStopsInDeclarationOrder(t *testing.T) {
	called := false
	s := validate.String().And(validate.String().Refine(func(string) error { return errors.New("first") })).
		Refine(func(string) error { called = true; return errors.New("later") })
	err := s.ValidateFirst("x")
	if called || err == nil {
		t.Fatalf("later rule ran=%v, err=%v", called, err)
	}
}

type cancelSchema struct {
	shape.Schema[int]
	cancel context.CancelFunc
}

func (s cancelSchema) TransformContext(context.Context, int) (int, error) { s.cancel(); return 1, nil }
func (s cancelSchema) ValidateContext(context.Context, int) error         { s.cancel(); return nil }
func (s cancelSchema) ValidateFirstContext(context.Context, int) error    { s.cancel(); return nil }

func TestExplicitLastCallbackCancellation(t *testing.T) {
	type DTO struct{ Value int }
	for _, mode := range []string{"transform", "validate", "first"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := shape.New[DTO](shape.Field("Value", cancelSchema{shape.Int(), cancel}))
			var err error
			switch mode {
			case "transform":
				_, err = s.TransformContext(ctx, DTO{})
			case "validate":
				err = s.ValidateContext(ctx, DTO{})
			case "first":
				err = s.ValidateFirstContext(ctx, DTO{})
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestTaggedPointerOuterWorkPrecedesInnerWork(t *testing.T) {
	type DTO struct {
		First  *string `shape:"ifnull=' guest ',trim"`
		Second *string `shape:"trim,ifnull=' guest '"`
	}
	out, err := shape.ParseJSON(shape.FromTags[DTO](), []byte(`{}`))
	if err != nil || *out.First != "guest" || *out.Second != "guest" {
		t.Fatalf("out=%v err=%v", out, err)
	}
}
