package transform_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rhevorn/shape/transform"
)

func TestStringFluent(t *testing.T) {
	got, err := transform.String().IfZero(" guest ").Trim().ToLower().Transform("")
	if err != nil || got != "guest" {
		t.Fatalf("got %q, %v", got, err)
	}
}
func TestFuncAndThen(t *testing.T) {
	boom := errors.New("boom")
	a := transform.Int().Apply(func(v int) (int, error) { return v + 1, nil }, func(v int) (int, error) { return v * 2, nil })
	b := transform.Int().Apply(func(v int) (int, error) {
		if v == 4 {
			return 0, boom
		}
		return v, nil
	})
	_, err := a.Then(b).Transform(1)
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
}

func TestThenUsesOneWorkingCopy(t *testing.T) {
	var firstStorage *int
	var secondStorage *int
	first := transform.Value[[]int]().Apply(func(value []int) ([]int, error) {
		firstStorage = &value[0]
		value[0]++
		return value, nil
	})
	second := transform.Value[[]int]().Apply(func(value []int) ([]int, error) {
		secondStorage = &value[0]
		value[0]++
		return value, nil
	})

	input := []int{1}
	out, err := first.Then(second).Transform(input)
	if err != nil {
		t.Fatal(err)
	}
	if input[0] != 1 || out[0] != 3 {
		t.Fatalf("input=%v output=%v", input, out)
	}
	if firstStorage == &input[0] || firstStorage != secondStorage {
		t.Fatalf("Then did not reuse one detached working copy: input=%p first=%p second=%p", &input[0], firstStorage, secondStorage)
	}
}
func TestCollectionsDoNotMutate(t *testing.T) {
	in := []string{" A "}
	out, err := transform.Slice(transform.String().Trim()).Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if in[0] != " A " || !reflect.DeepEqual(out, []string{"A"}) {
		t.Fatalf("in=%v out=%v", in, out)
	}
}
func TestMapCollision(t *testing.T) {
	_, err := transform.Map(transform.String().ToLower(), transform.Int()).Transform(map[string]int{"A": 1, "a": 2})
	if err == nil {
		t.Fatal("expected collision")
	}
}

func TestValueFuncCannotMutateInputSlice(t *testing.T) {
	in := []int{1}
	out, err := transform.Value[[]int]().Apply(func(v []int) ([]int, error) {
		v[0] = 2
		return v, nil
	}).Transform(in)
	if err != nil || in[0] != 1 || out[0] != 2 {
		t.Fatalf("in=%v out=%v err=%v", in, out, err)
	}
}

func TestCompositeMethodsRunInDeclarationOrder(t *testing.T) {
	fallback := 10
	firstFallback := transform.Pointer(transform.Int()).
		IfNull(&fallback).
		Apply(func(value *int) (*int, error) {
			out := *value + 1
			return &out, nil
		})
	got, err := firstFallback.Transform(nil)
	if err != nil || got == nil || *got != 11 {
		t.Fatalf("IfNull then Apply = %#v, %v", got, err)
	}

	firstApply := transform.Pointer(transform.Int()).
		Apply(func(value *int) (*int, error) {
			if value == nil {
				out := 20
				return &out, nil
			}
			return value, nil
		}).
		IfNull(&fallback)
	got, err = firstApply.Transform(nil)
	if err != nil || got == nil || *got != 20 {
		t.Fatalf("Apply then IfNull = %#v, %v", got, err)
	}
}
