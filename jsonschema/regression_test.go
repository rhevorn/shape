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
