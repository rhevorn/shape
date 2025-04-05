package validate

import (
	"context"

	"github.com/rhevorn/shape/internal/validationlocale"
)

// Language identifies a built-in validation-message language.
type Language = validationlocale.Language

const (
	// English selects English built-in messages.
	English = validationlocale.English
	// SimplifiedChinese selects Simplified Chinese built-in messages.
	SimplifiedChinese = validationlocale.SimplifiedChinese
)

// SetLanguage changes the process-wide default message language.
func SetLanguage(lang Language) { validationlocale.Set(requireLanguage(lang)) }

// WithLocale overrides the message language for operations using ctx.
func WithLocale(ctx context.Context, lang Language) context.Context {
	return validationlocale.With(ctx, requireLanguage(lang))
}

func languageFromContext(ctx context.Context) Language {
	return validationlocale.Get(ctx)
}

func requireLanguage(lang Language) Language {
	switch lang {
	case English, SimplifiedChinese:
		return lang
	default:
		panic("validate: unsupported language")
	}
}
