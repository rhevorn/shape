package validationlocale

import (
	"context"
	"sync/atomic"
)

type contextKey struct{}

// Language identifies a built-in validation-message language.
type Language uint8

const (
	English Language = iota
	SimplifiedChinese
)

var global atomic.Uint32

func Set(language Language) { global.Store(uint32(language)) }

func With(ctx context.Context, language Language) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, language)
}

func Get(ctx context.Context) Language {
	if ctx != nil {
		if language, ok := ctx.Value(contextKey{}).(Language); ok {
			return language
		}
	}
	return Language(global.Load())
}
