package jsonschema_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
)

func property[T any](t *testing.T, name string) map[string]any {
	t.Helper()
	doc, err := jsonschema.Export(shape.FromTags[T]())
	if err != nil {
		t.Fatal(err)
	}
	return rootObject(t, doc)["properties"].(map[string]any)[name].(map[string]any)
}

func TestIntersectConstraintsInEitherOrder(t *testing.T) {
	type A struct {
		V string `json:"v" shape:"minlength=5,notempty,len=7,maxlength=12"`
	}
	type B struct {
		V string `json:"v" shape:"maxlength=12,len=7,notempty,minlength=5"`
	}
	a, b := property[A](t, "v"), property[B](t, "v")
	if !reflect.DeepEqual(a, b) || a["minLength"] != 7 || a["maxLength"] != 7 {
		t.Fatalf("a=%v b=%v", a, b)
	}
	type Big struct {
		V uint64 `json:"v" shape:"min=18446744073709551614,nonnegative"`
	}
	p := property[Big](t, "v")
	if p["minimum"] != uint64(18446744073709551614) {
		t.Fatalf("uint64 rounded: %v", p)
	}
	type List struct {
		V []string `json:"v" shape:"min=5,notempty,len=7,max=12"`
	}
	p = property[List](t, "v")
	if p["minItems"] != 7 || p["maxItems"] != 7 {
		t.Fatalf("list=%v", p)
	}
}

func TestRepeatedEnumsAndPatternsKeepTheirConjunction(t *testing.T) {
	type DTO struct {
		V string `json:"v" shape:"oneof=ab|ac,oneof=ab|bc,pattern=^a,pattern=b$"`
	}
	p := property[DTO](t, "v")
	all, ok := p["allOf"].([]any)
	if !ok || len(all) != 2 {
		t.Fatalf("lost constraints: %v", p)
	}
	if !reflect.DeepEqual(all[0].(map[string]any)["enum"], []any{"ab", "bc"}) {
		t.Fatalf("enum=%v", all)
	}
	if all[1].(map[string]any)["pattern"] != `b(?![\s\S])` {
		t.Fatalf("pattern=%v", all)
	}
}

func TestUnrepresentableWireTypesFailExplicitly(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"quoted", func() error {
			type DTO struct {
				V int `json:"v,string"`
			}
			_, err := jsonschema.Export(shape.FromTags[DTO]())
			return err
		}},
		{"bytes", func() error {
			type DTO struct{ V []byte }
			_, err := jsonschema.Export(shape.FromTags[DTO]())
			return err
		}},
		{"named-bytes", func() error {
			type Bytes []byte
			type DTO struct{ V Bytes }
			_, err := jsonschema.Export(shape.FromTags[DTO]())
			return err
		}},
		{"unicode-case-fold", func() error {
			type DTO struct {
				V string `shape:"pattern='(?i)k'"`
			}
			_, err := jsonschema.Export(shape.FromTags[DTO]())
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var unsupported *jsonschema.UnsupportedError
			if err := tc.run(); !errors.As(err, &unsupported) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestExportRejectsDecodingDependentConstraints(t *testing.T) {
	tests := []struct {
		name   string
		export func() error
	}{
		{"float32", func() error {
			type Doc struct{ V float32 }
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
		{"float32 bound", func() error {
			type Doc struct {
				V float32 `shape:"min=0.1"`
			}
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
		{"float32 pointer", func() error {
			type Doc struct{ V *float32 }
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
		{"url", func() error {
			type Doc struct {
				V string `shape:"url"`
			}
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
		{"unique integer zero and null", func() error {
			type Doc struct {
				V []int `shape:"unique"`
			}
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
		{"unique string zero and null", func() error {
			type Doc struct {
				V []string `shape:"unique"`
			}
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
		{"unique struct defaults", func() error {
			type Item struct{ N int }
			type Doc struct {
				V []Item `shape:"unique"`
			}
			_, err := jsonschema.Export(shape.FromTags[Doc]())
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var unsupported *jsonschema.UnsupportedError
			if err := tc.export(); !errors.As(err, &unsupported) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestUniquePointerScalarsKeepNullDistinct(t *testing.T) {
	type Doc struct {
		V []*int `json:"v" shape:"unique,max=2000"`
	}
	schema := shape.FromTags[Doc]()
	p := rootObject(t, property[Doc](t, "v"))
	if p["uniqueItems"] != true || p["maxItems"] != 1024 {
		t.Fatalf("property=%v", p)
	}
	for _, tc := range []struct {
		input string
		valid bool
	}{{`{"v":[0,null]}`, true}, {`{"v":[0,0]}`, false}, {`{"v":[null,null]}`, false}} {
		if _, err := schema.ParseJSON([]byte(tc.input)); (err == nil) != tc.valid {
			t.Fatalf("input=%s error=%v", tc.input, err)
		}
	}
	tooMany := make([]*int, 1025)
	for i := range tooMany {
		tooMany[i] = new(int)
		*tooMany[i] = i
	}
	if schema.Validate(Doc{V: tooMany}) == nil {
		t.Fatal("deep equality bound missing")
	}
}
