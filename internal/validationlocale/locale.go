package validationlocale

import (
	"context"
	"sync/atomic"
)

type contextKey struct{}

var global atomic.Uint32

func Set(language uint8) { global.Store(uint32(language)) }

func With(ctx context.Context, language uint8) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, language)
}

func Get(ctx context.Context) uint8 {
	if ctx != nil {
		if language, ok := ctx.Value(contextKey{}).(uint8); ok {
			return language
		}
	}
	return uint8(global.Load())
}
