package transform_test

import (
	"context"
	"github.com/rhevorn/shape/transform"
	"testing"
)

type wrappedTransformer struct {
	*transform.ValueTransformer[string]
}

func (wrappedTransformer) TransformContext(context.Context, string) (string, error) {
	return "overridden", nil
}

func TestEmbeddedTransformerUsesPublicOverride(t *testing.T) {
	inner := wrappedTransformer{}
	for _, transformer := range []transform.Transformer[[]string]{transform.Slice[string](inner), transform.Slice[string](&inner)} {
		out, err := transformer.Transform([]string{"original"})
		if err != nil || out[0] != "overridden" {
			t.Fatalf("out=%v err=%v", out, err)
		}
	}
	sequence := transform.String().Then(inner)
	if out, err := sequence.Transform("original"); err != nil || out != "overridden" {
		t.Fatalf("out=%s err=%v", out, err)
	}
}
