package validate

import (
	"context"

	"github.com/rhevorn/shape/internal/validationlocale"
)

type Language uint8

const (
	English Language = iota
	SimplifiedChinese
)

func SetLanguage(lang Language) { validationlocale.Set(uint8(requireLanguage(lang))) }

func WithLocale(ctx context.Context, lang Language) context.Context {
	return validationlocale.With(ctx, uint8(requireLanguage(lang)))
}

func languageFromContext(ctx context.Context) Language {
	return Language(validationlocale.Get(ctx))
}

func requireLanguage(lang Language) Language {
	switch lang {
	case English, SimplifiedChinese:
		return lang
	default:
		panic("validate: unsupported language")
	}
}
