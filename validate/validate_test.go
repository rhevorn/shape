package validate_test

import (
	"context"
	"errors"
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
