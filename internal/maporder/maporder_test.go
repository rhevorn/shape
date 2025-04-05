package maporder

import (
	"context"
	"reflect"
	"testing"
)

func TestSortedSupportsEveryKeyFamily(t *testing.T) {
	type namedString string
	type namedInt int16
	type namedUint uint32

	strings, err := Sorted(context.Background(), map[namedString]int{"b": 2, "a": 1})
	if err != nil || !reflect.DeepEqual(strings, []namedString{"a", "b"}) {
		t.Fatalf("string keys = %v, %v", strings, err)
	}
	ints, err := Sorted(context.Background(), map[namedInt]int{2: 2, -1: 1})
	if err != nil || !reflect.DeepEqual(ints, []namedInt{-1, 2}) {
		t.Fatalf("signed keys = %v, %v", ints, err)
	}
	uints, err := Sorted(context.Background(), map[namedUint]int{2: 2, 1: 1})
	if err != nil || !reflect.DeepEqual(uints, []namedUint{1, 2}) {
		t.Fatalf("unsigned keys = %v, %v", uints, err)
	}
}

func TestSortedObservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Sorted(ctx, map[int]int{1: 1}); err != context.Canceled {
		t.Fatalf("error = %v", err)
	}
}
