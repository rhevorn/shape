package goshape

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

type treeNode struct {
	Value    string
	Children []treeNode
}

func treeSchema(calls *atomic.Int32) Schema[treeNode] {
	var schema Schema[treeNode]
	schema = Lazy("TreeNode", func() Schema[treeNode] {
		if calls != nil {
			calls.Add(1)
		}
		return Object[treeNode](
			Field("value", String().NonEmpty(), func(node *treeNode, value string) {
				node.Value = value
			}),
			Field("children", Slice(schema), func(node *treeNode, children []treeNode) {
				node.Children = children
			}).Default(nil),
		).Strict()
	})
	return schema
}

func TestLazyRecursiveParse(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	schema := treeSchema(&calls)
	input := map[string]any{
		"value": "root",
		"children": []any{
			map[string]any{"value": "leaf"},
		},
	}
	want := treeNode{Value: "root", Children: []treeNode{{Value: "leaf"}}}
	got, err := schema.Parse(input)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Lazy parse = %#v, %v; want %#v", got, err, want)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}

	issues := requireIssueCodes(t, parseError(schema, map[string]any{
		"value": "root",
		"children": []any{
			map[string]any{"value": ""},
		},
	}), CodeTooSmall)
	if got := issues[0].Path.String(); got != "children[0].value" {
		t.Fatalf("recursive issue path = %q", got)
	}
}

func TestLazyResolvesOnceConcurrently(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	schema := treeSchema(&calls)
	const workers = 32
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			if _, err := schema.Parse(map[string]any{"value": "node"}); err != nil {
				t.Errorf("Parse: %v", err)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
}

func TestLazyJSONSchemaDefinitions(t *testing.T) {
	t.Parallel()

	document, err := JSONSchema(treeSchema(nil))
	if err != nil {
		t.Fatal(err)
	}
	if document["$ref"] != "#/$defs/TreeNode" {
		t.Fatalf("root reference = %#v", document["$ref"])
	}
	definitions := document["$defs"].(map[string]any)
	definition := definitions["TreeNode"].(map[string]any)
	children := definition["properties"].(map[string]any)["children"].(map[string]any)
	items := children["items"].(map[string]any)
	if items["$ref"] != "#/$defs/TreeNode" {
		t.Fatalf("recursive reference = %#v", items)
	}
}

func TestLazyJSONSchemaEscapesNameAndRejectsDuplicates(t *testing.T) {
	t.Parallel()

	escaped, err := JSONSchema(Lazy("path~/node", func() Schema[string] { return String() }))
	if err != nil {
		t.Fatal(err)
	}
	if escaped["$ref"] != "#/$defs/path~0~1node" {
		t.Fatalf("escaped reference = %#v", escaped["$ref"])
	}

	first := Lazy("Entry", func() Schema[string] { return String() })
	second := Lazy("Entry", func() Schema[string] { return UUID() })
	if _, err := JSONSchema(Union[string](first, second)); err == nil {
		t.Fatal("duplicate lazy schema name was accepted")
	}
}

func TestLazyRejectsInvalidProviders(t *testing.T) {
	t.Parallel()

	requirePanic(t, func() { Lazy[string]("", func() Schema[string] { return String() }) })
	requirePanic(t, func() { Lazy[string]("Name", nil) })
	var nilSchema Schema[string]
	schema := Lazy("Name", func() Schema[string] { return nilSchema })
	if _, err := schema.Parse("value"); err == nil {
		t.Fatal("nil lazy provider result was accepted")
	}
	requirePanic(t, func() { Lazy("Name", func() Schema[string] { return String() }).MaxDepth(0) })
}

func TestLazyBoundsRecursiveDepthAndCycles(t *testing.T) {
	t.Parallel()

	type link struct{ Next *link }
	var schema LazySchema[link]
	schema = Lazy("BoundedLink", func() Schema[link] {
		return Object[link](
			Field("next", Nullable[link](schema), func(value *link, next *link) { value.Next = next }).Optional(),
		)
	}).MaxDepth(3)

	cycle := map[string]any{}
	cycle["next"] = cycle
	_, err := schema.Parse(cycle)
	issues := requireIssueCodes(t, err, CodeTooDeep)
	if got := issues[0].Path.String(); got != "next.next.next" {
		t.Fatalf("depth issue path = %q", got)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("depth error unexpectedly reported cancellation: %v", err)
	}
}
