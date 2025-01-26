package shape

import "context"

// Schema parses and validates an untrusted value into T.
//
// Implementations must be safe for concurrent calls after construction.
type Schema[T any] interface {
	Parse(value any) (T, error)
	ParseContext(ctx context.Context, value any) (T, error)
}
