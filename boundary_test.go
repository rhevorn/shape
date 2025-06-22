package shape_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
	"github.com/rhevorn/shape/validate"
)

type emptyErrorSchema struct{ shape.ValueSpec[string] }

func (emptyErrorSchema) ValidateContext(context.Context, string) error      { return &validate.Error{} }
func (emptyErrorSchema) ValidateFirstContext(context.Context, string) error { return &validate.Error{} }

func TestForeignFieldValidationErrorMustNotDisappear(t *testing.T) {
	type Doc struct{ Name string }
	schema := shape.New[Doc](shape.Field("Name", emptyErrorSchema{shape.Value[string]()}))
	if err := schema.Validate(Doc{}); err == nil {
		t.Error("Validate swallowed a non-nil validation error")
	}
	if err := schema.ValidateFirst(Doc{}); err == nil {
		t.Error("ValidateFirst swallowed a non-nil validation error")
	}
	if _, err := schema.ParseJSON([]byte(`{"Name":"bad"}`)); err == nil {
		t.Error("ParseJSON returned success after field validation failed")
	}
}

func TestMapRolesWithChildLabels(t *testing.T) {
	schema := shape.Map(shape.String().NotEmpty().Label("entry"), shape.String().NotEmpty().Label("entry"))
	var got *validate.Error
	if !errors.As(schema.Validate(map[string]string{"": ""}), &got) {
		t.Fatal("expected validation error")
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Issues []struct {
			Target string `json:"target"`
		}
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Issues[0].Target != "key" || wire.Issues[1].Target != "" {
		t.Fatalf("wire=%s", encoded)
	}
	a, b := got.Issues[0], got.Issues[1]
	if a.Target != validate.TargetKey || b.Target != validate.TargetValue || a.Label != "entry" || b.Label != "entry" {
		t.Errorf("key and value issues indistinguishable: %#v", got.Issues)
	}
}

type sharedItem struct {
	Name string `json:"name" shape:"trim"`
}
type sharedItems []sharedItem

var sharedDecodedItems = sharedItems{{Name: " original "}}

func (s *sharedItems) UnmarshalJSON([]byte) error { *s = sharedDecodedItems; return nil }
func TestCustomDecoderMustNotBreakAtomicBind(t *testing.T) {
	type Doc struct {
		Items sharedItems `json:"items"`
		Age   int         `json:"age" shape:"min=18"`
	}
	sharedDecodedItems = sharedItems{{Name: " original "}}
	target := Doc{Items: sharedDecodedItems, Age: 20}
	err := shape.BindJSON(&target, []byte(`{"items":[],"age":0}`))
	if err == nil {
		t.Fatal("expected age validation to fail")
	}
	if target.Items[0].Name != " original " {
		t.Errorf("failed BindJSON mutated target through decoder-owned storage: %#v", target)
	}
}

type cancelingSchema struct {
	shape.ValueSpec[string]
	cancel context.CancelFunc
}

func (s cancelingSchema) ValidateContext(context.Context, string) error { s.cancel(); return nil }
func TestPackageParseJSONMustObserveFinalCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schema := cancelingSchema{ValueSpec: shape.Value[string](), cancel: cancel}
	out, err := shape.ParseJSONContext(ctx, schema, []byte(`"ok"`))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("out=%q err=%v ctx.Err()=%v", out, err, ctx.Err())
	}
}

type embeddedDoc struct {
	Name string `json:"name"`
}
type overriddenSchema struct{ shape.TaggedSpec[embeddedDoc] }

func (s overriddenSchema) TransformContext(_ context.Context, v embeddedDoc) (embeddedDoc, error) {
	v.Name = "overridden"
	return v, nil
}
func (s overriddenSchema) ValidateContext(_ context.Context, v embeddedDoc) error {
	if v.Name != "allowed" {
		return errors.New("custom rejection")
	}
	return nil
}
func TestEmbeddedSchemaMustNotExportUndeclaredBehavior(t *testing.T) {
	schema := overriddenSchema{shape.FromTags[embeddedDoc]()}
	if _, err := jsonschema.Export[embeddedDoc](schema); err == nil {
		t.Error("export accepted custom overridden transform and validation without representing either")
	}
}

type overriddenTransformSchema struct{ shape.StructSpec[embeddedDoc] }

func (s overriddenTransformSchema) TransformContext(_ context.Context, v embeddedDoc) (embeddedDoc, error) {
	v.Name = "overridden"
	return v, nil
}
func TestPackageParseJSONMustCallOverriddenTransform(t *testing.T) {
	schema := overriddenTransformSchema{shape.New[embeddedDoc]()}
	out, err := shape.ParseJSON[embeddedDoc](schema, []byte(`{"name":"original"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "overridden" {
		t.Errorf("package ParseJSON bypassed TransformContext: %#v", out)
	}
}

type overriddenValueSchema struct{ *shape.ValueSpec[string] }

func (overriddenValueSchema) TransformContext(_ context.Context, _ string) (string, error) {
	return "overridden", nil
}

func TestCompositePreservesEmbeddedOverrides(t *testing.T) {
	inner := overriddenValueSchema{} // The embedded pointer need not be initialized by the override.
	out, err := shape.Slice[string](inner).Transform([]string{"original"})
	if err != nil || len(out) != 1 || out[0] != "overridden" {
		t.Fatalf("out=%v err=%v", out, err)
	}
	type Doc struct{ Name string }
	explicit := shape.New[Doc](shape.Field[string]("Name", inner))
	doc, err := explicit.Transform(Doc{Name: "original"})
	if err != nil || doc.Name != "overridden" {
		t.Fatalf("out=%v err=%v", doc, err)
	}
}

func TestJSONSchemaPointersAndWrappers(t *testing.T) {
	base := shape.FromTags[embeddedDoc]()
	for _, schema := range []shape.Schema[embeddedDoc]{base, &base} {
		out, err := shape.ParseJSON(schema, []byte(`{"name":"original"}`))
		if err != nil || out.Name != "original" {
			t.Fatalf("out=%v err=%v", out, err)
		}
		if _, err := jsonschema.Export(schema); err != nil {
			t.Fatal(err)
		}
	}
	wrapped := overriddenSchema{base}
	for _, schema := range []shape.Schema[embeddedDoc]{wrapped, &wrapped} {
		_, err := shape.ParseJSON(schema, []byte(`{"name":"allowed"}`))
		if err == nil {
			t.Fatal("wrapped validation was bypassed")
		}
		var unsupported *shape.UnsupportedSchemaError
		if _, err := jsonschema.Export(schema); !errors.As(err, &unsupported) {
			t.Fatalf("export=%v", err)
		}
	}
	explicit := overriddenTransformSchema{shape.New[embeddedDoc]()}
	for _, schema := range []shape.Schema[embeddedDoc]{explicit, &explicit} {
		out, err := shape.ParseJSON(schema, []byte(`{"name":"original"}`))
		if err != nil || out.Name != "overridden" {
			t.Fatalf("out=%v err=%v", out, err)
		}
	}
}

func TestNestedMapKeyTargetSurvivesOuterValueValidation(t *testing.T) {
	schema := shape.Map(shape.String(), shape.Map(shape.String().NotEmpty().Label("entry"), shape.String().NotEmpty().Label("entry")))
	var got *validate.Error
	if !errors.As(schema.Validate(map[string]map[string]string{"outer": {"": ""}}), &got) {
		t.Fatal("missing validation error")
	}
	if len(got.Issues) != 2 || got.Issues[0].Target != validate.TargetKey || got.Issues[1].Target != validate.TargetValue {
		t.Fatalf("issues=%#v", got.Issues)
	}
	if got.Issues[0].Path.String() != `["outer"][""]` {
		t.Fatalf("path=%s", got.Issues[0].Path.String())
	}
}

func TestNamedPointerTags(t *testing.T) {
	type NamePtr *string
	type Doc struct {
		Name NamePtr `json:"name" shape:"ifnull=' guest ',trim,notnull"`
	}
	schema := shape.FromTags[Doc]()
	for _, input := range []string{`{}`, `{"name":null}`, `{"name":" guest "}`} {
		out, err := schema.ParseJSON([]byte(input))
		if err != nil || out.Name == nil || *out.Name != "guest" {
			t.Fatalf("input=%s out=%v err=%v", input, out, err)
		}
	}
}

type sharedTextValue struct{ Items []int }

var sharedTextItems = []int{1}

func (v *sharedTextValue) UnmarshalText([]byte) error { v.Items = sharedTextItems; return nil }

func TestTextDecoderOwnershipBeforeWholeTransform(t *testing.T) {
	type Doc struct {
		Value sharedTextValue `json:"value"`
	}
	sharedTextItems = []int{1}
	change := func(v Doc) (Doc, error) { v.Value.Items[0] = 2; return v, errors.New("failed") }
	for _, schema := range []shape.Schema[Doc]{shape.FromTags[Doc]().Apply(change), shape.New[Doc]().Apply(change)} {
		if _, err := shape.ParseJSON(schema, []byte(`{"value":"cached"}`)); err == nil {
			t.Fatal("expected transform failure")
		}
		if sharedTextItems[0] != 1 {
			t.Fatal("transform changed decoder-owned storage")
		}
	}
}

func TestNoopCompositeStillDetachesMutableValues(t *testing.T) {
	input := [][]int{{1}}
	out, err := shape.Slice(shape.Slice(shape.Int())).Transform(input)
	if err != nil {
		t.Fatal(err)
	}
	out[0][0] = 2
	if input[0][0] != 1 {
		t.Fatal("identity optimization shared caller storage")
	}
}

func TestConcurrentParsingWithSharedDecoderStorage(t *testing.T) {
	type Doc struct {
		Items sharedItems `json:"items"`
	}
	sharedDecodedItems = sharedItems{{Name: " original "}}
	schema := shape.FromTags[Doc]()
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				out, err := schema.ParseJSON([]byte(`{"items":[]}`))
				if err != nil {
					t.Error(err)
					return
				}
				if out.Items[0].Name != "original" {
					t.Errorf("name=%q", out.Items[0].Name)
				}
				out.Items[0].Name = "changed"
			}
		}()
	}
	wg.Wait()
	if sharedDecodedItems[0].Name != " original " {
		t.Fatal("shared decoder storage was changed")
	}
}
