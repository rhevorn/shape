package transform_test

import (
	"strings"
	"testing"

	"github.com/rhevorn/shape/transform"
)

func FuzzStringTransformer(f *testing.F) {
	f.Add("  Pong  ")
	f.Add("\x00Go\n")
	t := transform.String().Trim().ToLower()
	f.Fuzz(func(tst *testing.T, input string) {
		out, err := t.Transform(input)
		if err != nil {
			tst.Fatal(err)
		}
		if out != strings.ToLower(strings.TrimSpace(input)) {
			tst.Fatalf("output = %q", out)
		}
	})
}
