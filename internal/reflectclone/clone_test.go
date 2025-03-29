package reflectclone_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/rhevorn/shape/internal/reflectclone"
)

func TestIsZeroTreatsZeroTimeAsZero(t *testing.T) {
	zero := time.Time{}.Local()
	if !zero.IsZero() {
		t.Fatal("precondition: time.Time{}.Local() should report itself as zero")
	}
	if !reflectclone.IsZero(reflect.ValueOf(zero)) {
		t.Fatal("IsZero disagreed with time.Time.IsZero")
	}
	if reflectclone.IsZero(reflect.ValueOf(time.Now())) {
		t.Fatal("IsZero reported a non-zero time as zero")
	}
}

func TestCloneDetachesUnexportedStorage(t *testing.T) {
	// The promoted-field case needs no unexported access to reach the private
	// backing array, which is what made it reachable from a user callback.
	type named struct{ Items []string }
	type promoted struct {
		named
		Public []int
	}

	t.Run("addressable", func(t *testing.T) {
		input := promoted{named: named{Items: []string{"orig"}}, Public: []int{1}}
		out, err := reflectclone.Clone(context.Background(), reflect.ValueOf(&input).Elem(), false)
		if err != nil {
			t.Fatal(err)
		}
		out.Interface().(promoted).Items[0] = "mutated"
		if input.Items[0] != "orig" {
			t.Fatalf("clone shared the private backing array: %#v", input.Items)
		}
	})

	t.Run("unaddressable source", func(t *testing.T) {
		// A struct reached through a map value has no addressable storage of
		// its own, so addressability cannot be assumed.
		m := map[string]promoted{"k": {named: named{Items: []string{"orig"}}, Public: []int{1}}}
		out, err := reflectclone.Clone(context.Background(), reflect.ValueOf(m), false)
		if err != nil {
			t.Fatal(err)
		}
		out.Interface().(map[string]promoted)["k"].Items[0] = "mutated"
		if m["k"].Items[0] != "orig" {
			t.Fatalf("clone shared a map value's private backing array: %#v", m["k"].Items)
		}
	})
}

func TestCloneBoundsRecursion(t *testing.T) {
	node := &struct{ Next any }{}
	cur := node
	for i := 0; i < reflectclone.MaxDepth+2; i++ {
		next := &struct{ Next any }{}
		cur.Next = next
		cur = next
	}
	if _, err := reflectclone.Clone(context.Background(), reflect.ValueOf(node), false); !errors.Is(err, reflectclone.ErrDepthExceeded) {
		t.Fatalf("deep value error = %v, want ErrDepthExceeded", err)
	}
}

// Strict mode backs Schema fallbacks, which are stored and reused across
// calls, so a type that cannot be snapshotted must be refused rather than
// silently shared.
func TestStrictModeRefusesUnsnapshotableTypes(t *testing.T) {
	fn := func() {}
	if _, err := reflectclone.Clone(context.Background(), reflect.ValueOf(fn), true); !errors.Is(err, reflectclone.ErrUnsupportedType) {
		t.Fatalf("strict func error = %v, want ErrUnsupportedType", err)
	}
	if _, err := reflectclone.Clone(context.Background(), reflect.ValueOf(fn), false); err != nil {
		t.Fatalf("non-strict func should be shared, got %v", err)
	}
}
