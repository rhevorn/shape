package validate_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/rhevorn/shape/validate"
)

func TestLanguageConstants(t *testing.T) {
	defer validate.SetLanguage(validate.English)

	validate.SetLanguage(validate.SimplifiedChinese)
	err := validate.String().NotEmpty().Validate("")
	var got *validate.Error
	if !errors.As(err, &got) || got.Issues[0].Message != "不能为空" {
		t.Fatalf("global language error = %#v", err)
	}
	ctx := validate.WithLocale(context.Background(), validate.English)
	err = validate.String().NotEmpty().ValidateContext(ctx, "")
	if !errors.As(err, &got) || got.Issues[0].Message != "must not be empty" {
		t.Fatalf("context language error = %#v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("invalid language did not panic")
		}
	}()
	validate.SetLanguage(validate.Language(255))
}

func TestValidateAllAndFirst(t *testing.T) {
	v := validate.String().NotEmpty().MinLength(2).Pattern(`^[a-z]+$`)
	err := v.Validate("")
	var all *validate.Error
	if !errors.As(err, &all) || len(all.Issues) != 3 {
		t.Fatalf("all = %#v", err)
	}
	err = v.ValidateFirst("")
	var first *validate.Error
	if !errors.As(err, &first) || len(first.Issues) != 1 {
		t.Fatalf("first = %#v", err)
	}
}

func TestRefineMultipleAndAnd(t *testing.T) {
	a := validate.String().Refine(
		func(string) error { return errors.New("a") },
		func(string) error { return errors.New("b") },
	)
	v := validate.String().NotEmpty().And(a)
	err := v.Validate("")
	var got *validate.Error
	if !errors.As(err, &got) || len(got.Issues) != 3 {
		t.Fatalf("issues = %#v", err)
	}
	if got.Issues[1].Code != validate.CodeCustom || got.Issues[1].Message != "a" {
		t.Fatalf("custom = %#v", got.Issues[1])
	}
}

func TestCollections(t *testing.T) {
	v := validate.Slice(validate.String().NotEmpty()).Len(2).Unique()
	if err := v.Validate([]string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := v.Validate([]string{"a", ""}); err == nil {
		t.Fatal("expected element error")
	}
	m := validate.Map(validate.String().NotEmpty(), validate.Int().Positive()).Len(1)
	if err := m.Validate(map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
}

// A NaN bound made both comparisons false, turning Between into a silent
// no-op that accepted every value.
func TestBetweenRejectsNaNWithoutRejectingInfinity(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func()
	}{
		{"NaN lower bound", func() { validate.Float64().Between(math.NaN(), 10) }},
		{"NaN upper bound", func() { validate.Float64().Between(0, math.NaN()) }},
		{"inverted bounds", func() { validate.Float64().Between(10, 0) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected a construction panic")
				}
			}()
			tc.call()
		})
	}
	// An infinite bound is a meaningful open-ended range and stays legal.
	if err := validate.Float64().Between(0, math.Inf(1)).Validate(5); err != nil {
		t.Fatalf("Between(0, +Inf) = %v", err)
	}
}

// A float reaching a schema must be rejected identically whether it was
// declared through Number[N] or through the generic Value[T] escape hatch.
func TestValueAndNumberAgreeOnNonFiniteFloats(t *testing.T) {
	for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		var valueErr, numberErr *validate.Error
		if !errors.As(validate.Value[float64]().Validate(x), &valueErr) {
			t.Fatalf("Value[float64] accepted %v", x)
		}
		if !errors.As(validate.Float64().Validate(x), &numberErr) {
			t.Fatalf("Float64 accepted %v", x)
		}
		if valueErr.Issues[0].Code != validate.CodeInvalidNumber || numberErr.Issues[0].Code != validate.CodeInvalidNumber {
			t.Fatalf("x=%v codes = %q, %q", x, valueErr.Issues[0].Code, numberErr.Issues[0].Code)
		}
	}
}

// The deep-uniqueness bound is a resource limit, not a verdict about the data,
// so it must not be reported under the length-violation code.
func TestUniqueDeepLimitUsesItsOwnCode(t *testing.T) {
	items := make([][]int, validate.MaxDeepUniqueItems+1)
	for i := range items {
		items[i] = []int{i}
	}
	v := validate.Slice(validate.Value[[]int]()).Unique()

	belowBound := items[:validate.MaxDeepUniqueItems]
	if err := v.Validate(belowBound); err != nil {
		t.Fatalf("distinct collection below the bound = %v", err)
	}

	var got *validate.Error
	if !errors.As(v.Validate(items), &got) || len(got.Issues) != 1 {
		t.Fatalf("over-bound error = %v", got)
	}
	if code := got.Issues[0].Code; code != validate.CodeUniqueLimit {
		t.Fatalf("Code = %q, want %q (must not be %q)", code, validate.CodeUniqueLimit, validate.CodeTooBig)
	}
}

// The same call shape must label issues identically in every family. The
// collection families used to build a fresh zero base for And and drop the
// receiver's label, while String/Number/Value kept it.
func TestAndKeepsLabelInEveryFamily(t *testing.T) {
	fail := errors.New("boom")
	issues := func(err error) []validate.Issue {
		t.Helper()
		var got *validate.Error
		if !errors.As(err, &got) {
			t.Fatalf("error = %v", err)
		}
		return got.Issues
	}

	stringErr := validate.String().Label("nick").
		And(validate.Value[string]().Refine(func(string) error { return fail })).
		Validate("x")
	if got := issues(stringErr)[0].Label; got != "nick" {
		t.Fatalf("string label = %q", got)
	}

	sliceErr := validate.Slice(validate.Value[string]()).Label("items").
		And(validate.Value[[]string]().Refine(func([]string) error { return fail })).
		Validate([]string{"x"})
	if got := issues(sliceErr)[0].Label; got != "items" {
		t.Fatalf("slice label = %q", got)
	}

	pointerErr := validate.Pointer(validate.Value[string]()).Label("head").
		And(validate.Value[*string]().Refine(func(*string) error { return fail })).
		Validate(new(string))
	if got := issues(pointerErr)[0].Label; got != "head" {
		t.Fatalf("pointer label = %q", got)
	}
}

// A map key failure and a map value failure share one path segment, so they
// used to produce byte-identical issues.
func TestMapKeyAndValueIssuesAreDistinguishable(t *testing.T) {
	err := validate.Map(validate.String().NotEmpty(), validate.String().NotEmpty()).
		Validate(map[string]string{"": ""})
	var got *validate.Error
	if !errors.As(err, &got) || len(got.Issues) != 2 {
		t.Fatalf("issues = %#v", err)
	}
	if got.Issues[0].Label != "key" || got.Issues[1].Label != "value" {
		t.Fatalf("labels = %q, %q", got.Issues[0].Label, got.Issues[1].Label)
	}
}

// The exported validators are structs whose zero value is writable from
// outside the package. It must report that as a configuration mistake for any
// input, rather than dereferencing nil for some values and passing others.
func TestZeroValueValidatorsReportMisconfiguration(t *testing.T) {
	value := 1
	for _, tc := range []struct {
		name string
		run  func()
	}{
		{"slice with items", func() { _ = validate.SliceValidator[int]{}.Validate([]int{1}) }},
		{"slice empty", func() { _ = validate.SliceValidator[int]{}.Validate(nil) }},
		{"map with entries", func() { _ = validate.MapValidator[string, int]{}.Validate(map[string]int{"a": 1}) }},
		{"pointer nil", func() { _ = validate.PointerValidator[int]{}.Validate(nil) }},
		{"pointer non-nil", func() { _ = validate.PointerValidator[int]{}.Validate(&value) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				cause := fmt.Sprint(recover())
				if !strings.Contains(cause, "must be built with validate.") {
					t.Fatalf("panic = %s", cause)
				}
			}()
			tc.run()
		})
	}
}

// A custom validator that ignores the context must still be stopped by the
// collection loop, matching Map's behaviour.
type ctxIgnoring struct{ calls *int }

func (c ctxIgnoring) Validate(string) error { return nil }
func (c ctxIgnoring) ValidateContext(context.Context, string) error {
	*c.calls++
	return nil
}
func (c ctxIgnoring) ValidateFirst(string) error { return nil }
func (c ctxIgnoring) ValidateFirstContext(context.Context, string) error {
	*c.calls++
	return nil
}

func TestSliceElementLoopStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	_ = validate.Slice[string](ctxIgnoring{calls: &calls}).ValidateContext(ctx, []string{"a", "b", "c"})
	if calls != 0 {
		t.Fatalf("inner validator ran %d times after cancellation", calls)
	}
}

func TestCompositeAndKeepsInnerValidation(t *testing.T) {
	v := validate.Slice(validate.String().NotEmpty()).And(
		validate.Value[[]string]().Refine(func([]string) error { return errors.New("extra") }),
	)
	err := v.Validate([]string{""})
	var got *validate.Error
	if !errors.As(err, &got) || len(got.Issues) != 2 {
		t.Fatalf("issues = %#v", err)
	}
}

func TestMapValidateFirstDoesNotRunValueAfterKeyFailure(t *testing.T) {
	calls := 0
	v := validate.Map(
		validate.String().NotEmpty(),
		validate.Int().Refine(func(int) error { calls++; return errors.New("value") }),
	)
	err := v.ValidateFirst(map[string]int{"": 1})
	if err == nil || calls != 0 {
		t.Fatalf("err=%v value calls=%d", err, calls)
	}
}

func TestLanguageIsAppliedWhenErrorIsCreated(t *testing.T) {
	defer validate.SetLanguage(validate.English)
	validate.SetLanguage(validate.SimplifiedChinese)

	err := validate.String().NotEmpty().Refine(func(string) error {
		return errors.New("business message")
	}).Validate("")
	var got *validate.Error
	if !errors.As(err, &got) || len(got.Issues) != 2 {
		t.Fatalf("error = %#v", err)
	}
	if got.Issues[0].Message != "不能为空" || got.Issues[1].Message != "business message" {
		t.Fatalf("issues = %#v", got.Issues)
	}

	ctx := validate.WithLocale(context.Background(), validate.English)
	err = validate.String().NotEmpty().ValidateContext(ctx, "")
	if !errors.As(err, &got) || got.Issues[0].Message != "must not be empty" {
		t.Fatalf("context error = %#v", err)
	}
}
