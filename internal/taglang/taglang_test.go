package taglang

import "testing"

func TestUnicodeWhitespaceIsHandledConsistently(t *testing.T) {
	items, err := Parse("trim\u00a0,\u00a0maxlength\u00a0=\u00a03")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Name != "trim" || items[1].Name != "maxlength" || items[1].Value != "3" {
		t.Fatalf("items = %#v", items)
	}
}
