package transform_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

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

// A private slice or map used to stay aliased to the caller's storage, because
// the clone skipped fields it could not Set. The promoted-field case needs no
// unexported access at all to reach that storage from a callback.
func TestCloneDetachesUnexportedFields(t *testing.T) {
	t.Run("promoted field over embedded private storage", func(t *testing.T) {
		type Doc struct {
			named
			Public []int
		}
		input := Doc{named: named{Items: []string{"orig"}}, Public: []int{1}}
		if _, err := transform.Value[Doc]().Apply(func(v Doc) (Doc, error) {
			v.Items[0] = "mutated"
			return v, nil
		}).Transform(input); err != nil {
			t.Fatal(err)
		}
		if input.Items[0] != "orig" {
			t.Fatalf("caller storage reached through a promoted field: %#v", input.Items)
		}
	})

	t.Run("direct private field", func(t *testing.T) {
		input := private{Public: []int{1}, hidden: []int{7}}
		if _, err := transform.Value[private]().Apply(func(v private) (private, error) {
			v.hidden[0] = 99
			return v, nil
		}).Transform(input); err != nil {
			t.Fatal(err)
		}
		if input.hidden[0] != 7 {
			t.Fatalf("private backing array was shared with the caller: %#v", input.hidden)
		}
	})
}

type named struct{ Items []string }

type private struct {
	Public []int
	hidden []int
}

// reflect.Value.IsZero and time.Time.IsZero disagree for a zero instant that
// carries a non-nil Location, so the fallback used to be skipped.
func TestIfZeroTreatsAZeroTimeAsZero(t *testing.T) {
	zero := time.Time{}.Local()
	if !zero.IsZero() || reflect.ValueOf(zero).IsZero() {
		t.Skip("this platform does not distinguish the two zero checks")
	}
	fallback := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := transform.Time().IfZero(fallback).Transform(zero)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(fallback) {
		t.Fatalf("IfZero returned %v, want the fallback %v", got, fallback)
	}
}

// cachedTransformer is a foreign Transformer: it makes no promise about where
// the value it returns lives.
type cachedTransformer struct{ cached []int }

func (c cachedTransformer) Transform([]int) ([]int, error) { return c.cached, nil }
func (c cachedTransformer) TransformContext(context.Context, []int) ([]int, error) {
	return c.cached, nil
}

// A foreign transformer's output must be detached before later steps treat it
// as their private working value, or they mutate storage it still holds.
func TestForeignTransformerOutputIsDetached(t *testing.T) {
	foreign := cachedTransformer{cached: []int{1, 2, 3}}
	_, err := transform.Value[[]int]().
		Then(foreign, transform.Slice(transform.Int().Apply(func(v int) (int, error) { return v + 1, nil }))).
		Transform([]int{9})
	if err != nil {
		t.Fatal(err)
	}
	if foreign.cached[0] != 1 {
		t.Fatalf("foreign transformer's own storage was mutated: %v", foreign.cached)
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
