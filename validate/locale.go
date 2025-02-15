package validate

import (
	"context"
	"errors"
	"sync/atomic"
)

type localeKey struct{}

type Language uint8

const (
	English Language = iota
	SimplifiedChinese
)

var language atomic.Value

func init() { language.Store(English) }

func SetLanguage(lang Language) { language.Store(requireLanguage(lang)) }

func CurrentLanguage() Language { return language.Load().(Language) }

func WithLocale(ctx context.Context, lang Language) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, localeKey{}, requireLanguage(lang))
}

func LocaleFromContext(ctx context.Context) Language {
	if ctx != nil {
		if lang, ok := ctx.Value(localeKey{}).(Language); ok {
			return lang
		}
	}
	return CurrentLanguage()
}

func Localize(err error, lang Language) error {
	return localizeError(err, requireLanguage(lang))
}

func LocalizeContext(ctx context.Context, err error) error {
	return localizeError(err, LocaleFromContext(ctx))
}

func requireLanguage(lang Language) Language {
	switch lang {
	case English, SimplifiedChinese:
		return lang
	default:
		panic("validate: unsupported language")
	}
}

func localizeError(err error, lang Language) error {
	var ve *Error
	if !errors.As(err, &ve) || ve == nil {
		return err
	}
	out := make([]Issue, len(ve.Issues))
	for i, issue := range ve.Issues {
		out[i] = localizeIssue(issue, lang)
	}
	return &Error{Issues: out}
}
