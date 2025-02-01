package shape

import (
	"context"
	"strings"
	"sync/atomic"
)

type localeContextKey struct{}

var processLanguage atomic.Value // stores string

func init() {
	processLanguage.Store("")
}

// SetLanguage sets the process-wide default language used when a parse
// context does not carry a locale. Pass an empty string to reset to English.
//
// Prefer WithLocale for per-request language selection. SetLanguage is intended
// for single-language applications that configure language once at startup.
func SetLanguage(lang string) {
	processLanguage.Store(strings.TrimSpace(lang))
}

// Language returns the process-wide default language, or empty when unset.
func Language() string {
	value, _ := processLanguage.Load().(string)
	return value
}

// WithLocale stores a language on ctx for ParseContext and related helpers.
// A non-empty context locale overrides SetLanguage.
func WithLocale(ctx context.Context, lang string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, localeContextKey{}, strings.TrimSpace(lang))
}

// LocaleFromContext returns the language stored by WithLocale, if any.
func LocaleFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(localeContextKey{}).(string)
	return value
}

func resolveLocale(ctx context.Context) string {
	if lang := LocaleFromContext(ctx); lang != "" {
		return normalizeLocale(lang)
	}
	if lang := Language(); lang != "" {
		return normalizeLocale(lang)
	}
	return "en"
}

func normalizeLocale(lang string) string {
	lang = strings.TrimSpace(lang)
	lang = strings.ReplaceAll(lang, "_", "-")
	switch {
	case lang == "":
		return "en"
	case strings.EqualFold(lang, "en"), strings.EqualFold(lang, "en-US"), strings.EqualFold(lang, "en-GB"):
		return "en"
	case strings.EqualFold(lang, "zh"), strings.EqualFold(lang, "zh-CN"), strings.EqualFold(lang, "zh-Hans"):
		return "zh-CN"
	default:
		return lang
	}
}
