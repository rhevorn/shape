package shape

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

type cancellationCounterContext struct {
	context.Context
	calls  int
	cancel int
}

func (c *cancellationCounterContext) Err() error {
	c.calls++
	if c.calls >= c.cancel {
		return context.Canceled
	}
	return nil
}

type uncheckedSliceSchema struct{}

func (uncheckedSliceSchema) Parse(value any) ([]int, error) { return value.([]int), nil }
func (uncheckedSliceSchema) ParseContext(_ context.Context, value any) ([]int, error) {
	return value.([]int), nil
}

func TestCompositeSchemasPropagateCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name string
		run  func() error
	}{
		{"slice", func() error { _, err := Slice(String()).ParseContext(ctx, []any{"x"}); return err }},
		{"map", func() error { _, err := Map(String(), String()).ParseContext(ctx, map[string]any{"x": "y"}); return err }},
		{"map-keyed", func() error { _, err := Map(String(), Int()).ParseContext(ctx, map[string]any{"x": 1}); return err }},
		{"tuple", func() error { _, err := coordinateSchema().ParseContext(ctx, []any{1, 2, "x"}); return err }},
		{"object", func() error { _, err := testUserSchema().ParseContext(ctx, map[string]any{}); return err }},
		{"union", func() error { _, err := Union[string](String(), UUID()).ParseContext(ctx, "x"); return err }},
		{"oneOf", func() error { _, err := OneOf[string](String().Email(), UUID()).ParseContext(ctx, "x"); return err }},
		{"nullable", func() error { _, err := Nullable(String()).ParseContext(ctx, nil); return err }},
		{"lazy", func() error { _, err := treeSchema(nil).ParseContext(ctx, map[string]any{}); return err }},
		{"transform", func() error {
			_, err := Transform(String(), func(string) (int, error) { return 1, nil }).ParseContext(ctx, "x")
			return err
		}},
		{"refine", func() error {
			_, err := Refine(String(), func(string) error { return nil }).ParseContext(ctx, "x")
			return err
		}},
		{"context refine", func() error {
			_, err := RefineContext(String(), func(context.Context, string) error { return nil }).ParseContext(ctx, "x")
			return err
		}},
		{"annotated", func() error { _, err := Annotate(String()).ParseContext(ctx, "x"); return err }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestUniqueChecksCancellationDuringFallback(t *testing.T) {
	t.Parallel()

	input := make([]any, 100)
	for index := range input {
		input[index] = []int{index}
	}
	ctx := &cancellationCounterContext{Context: context.Background(), cancel: len(input) + 2}
	if _, err := Slice[[]int](uncheckedSliceSchema{}).Unique().ParseContext(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unique error = %v, want context.Canceled", err)
	}
	requireIssueCodes(t, parseError(
		Slice[[]int](uncheckedSliceSchema{}).Unique(),
		[][]int{{1, 2}, {1, 2}},
	), CodeInvalidValue)
	tooMany := make([][]int, DefaultMaxDeepUniqueItems+1)
	for index := range tooMany {
		tooMany[index] = []int{index}
	}
	requireIssueCodes(t, parseError(
		Slice[[]int](uncheckedSliceSchema{}).Unique(),
		tooMany,
	), CodeTooBig)
}

func TestCompositeIssueAggregationIsBounded(t *testing.T) {
	t.Parallel()

	input := make([]any, DefaultMaxIssues+50)
	for index := range input {
		input[index] = index
	}
	issues := requireIssues(t, parseError(Slice(String()), input))
	if len(issues) != DefaultMaxIssues {
		t.Fatalf("issue count = %d, want %d", len(issues), DefaultMaxIssues)
	}
	if got := issues[len(issues)-1].Code; got != CodeTooManyIssues {
		t.Fatalf("last issue = %q, want %q", got, CodeTooManyIssues)
	}
}

func TestCompositeIssueOrderIsDeterministic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		err   error
		paths []string
	}{
		{
			name:  "slice",
			err:   parseError(Slice(String().Min(1)), []any{"", ""}),
			paths: []string{"[0]", "[1]"},
		},
		{
			name:  "map",
			err:   parseError(Map(String(), String().Min(1)), map[string]any{"z": "", "a": ""}),
			paths: []string{"a", "z"},
		},
		{
			name:  "record",
			err:   parseError(Map(String(), String().Min(1)), map[string]any{"z": "", "a": ""}),
			paths: []string{"a", "z"},
		},
		{
			name:  "tuple",
			err:   parseError(coordinateSchema(), []any{"bad", "bad", ""}),
			paths: []string{"[0]", "[1]", "[2]"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var validation *ValidationError
			if !errors.As(test.err, &validation) {
				t.Fatalf("error = %T, want *ValidationError", test.err)
			}
			paths := make([]string, len(validation.Issues))
			for index, issue := range validation.Issues {
				paths[index] = issue.Path.String()
			}
			if !reflect.DeepEqual(paths, test.paths) {
				t.Fatalf("paths = %v, want %v", paths, test.paths)
			}
		})
	}

	type target struct{}
	object := Object[target](
		Field("z", String(), func(*target, string) {}),
		Field("a", String(), func(*target, string) {}),
	).Strict()
	var validation *ValidationError
	err := parseError(object, map[string]any{"c": true, "b": true})
	if !errors.As(err, &validation) {
		t.Fatalf("object error = %T", err)
	}
	paths := make([]string, len(validation.Issues))
	for index, issue := range validation.Issues {
		paths[index] = issue.Path.String()
	}
	if want := []string{"z", "a", "b", "c"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("object paths = %v, want %v", paths, want)
	}
}

func TestCompositeBuildersAreImmutable(t *testing.T) {
	t.Parallel()

	baseSlice := Slice(String())
	if _, err := baseSlice.Max(0).Parse([]any{"x"}); err == nil {
		t.Fatal("bounded slice accepted oversized input")
	}
	if _, err := baseSlice.Parse([]any{"x"}); err != nil {
		t.Fatalf("derived slice mutated base: %v", err)
	}

	baseMap := Map(String(), String())
	if _, err := baseMap.Max(0).Parse(map[string]any{"x": "y"}); err == nil {
		t.Fatal("bounded map accepted oversized input")
	}
	if _, err := baseMap.Parse(map[string]any{"x": "y"}); err != nil {
		t.Fatalf("derived map mutated base: %v", err)
	}

	baseMapKeyed := Map(String(), Int())
	if _, err := baseMapKeyed.Max(0).Parse(map[string]any{"x": 1}); err == nil {
		t.Fatal("bounded record accepted oversized input")
	}
	if _, err := baseMapKeyed.Parse(map[string]any{"x": 1}); err != nil {
		t.Fatalf("derived record mutated base: %v", err)
	}

	baseTuple := coordinateSchema()
	rejected := baseTuple.Refine(func(coordinate) error { return errors.New("rejected") })
	if _, err := rejected.Parse([]any{1, 2, "x"}); err == nil {
		t.Fatal("refined tuple accepted input")
	}
	if _, err := baseTuple.Parse([]any{1, 2, "x"}); err != nil {
		t.Fatalf("derived tuple mutated base: %v", err)
	}

	alternatives := []Schema[string]{String()}
	union := Union(alternatives...)
	alternatives[0] = nil
	if _, err := union.Parse("x"); err != nil {
		t.Fatalf("union retained caller slice: %v", err)
	}
}

func TestCompositeSchemasConcurrentReuse(t *testing.T) {
	t.Parallel()

	recursive := treeSchema(nil)
	record := Map(String().ToLower(), Int().Positive())
	tuple := coordinateSchema()
	union := OneOf[string](String().Email(), UUID())
	const workers = 24
	const iterations = 40
	errorsFound := make(chan error, workers)
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			for range iterations {
				if _, err := recursive.Parse(map[string]any{"value": "node"}); err != nil {
					errorsFound <- err
					return
				}
				if _, err := ExportDocument(recursive); err != nil {
					errorsFound <- err
					return
				}
				if _, err := record.Parse(map[string]any{"ONE": 1}); err != nil {
					errorsFound <- err
					return
				}
				if _, err := tuple.Parse([]any{1, 2, "point"}); err != nil {
					errorsFound <- err
					return
				}
				if _, err := union.Parse("user@example.com"); err != nil {
					errorsFound <- err
					return
				}
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}
