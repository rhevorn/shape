package transform

import (
	"context"
	"errors"
	"reflect"

	"github.com/rhevorn/shape/internal/reflectclone"
)

func clone[T any](v T) (T, error) { return cloneContext(context.Background(), v) }

// cloneContext detaches v from caller-owned storage. Built-in steps receive
// this private working value and may mutate it in place.
//
// The implementation lives in internal/reflectclone so the root package and
// transform cannot drift apart on what "detached" means.
func cloneContext[T any](ctx context.Context, v T) (T, error) {
	var zero T
	out, err := reflectclone.Clone(ctx, reflect.ValueOf(&v).Elem(), false)
	if err != nil {
		return zero, cloneError(err)
	}
	return out.Interface().(T), nil
}

func cloneError(err error) error {
	if errors.Is(err, reflectclone.ErrDepthExceeded) {
		return errors.New("transform: copy depth exceeded")
	}
	return err
}
