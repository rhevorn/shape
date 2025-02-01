package shape

import (
	"context"
	"errors"
	"testing"
)

func TestSetLanguageLocalizesMessages(t *testing.T) {
	previous := Language()
	t.Cleanup(func() { SetLanguage(previous) })

	SetLanguage("zh-CN")
	_, err := String().Min(3).Parse("x")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if got := validation.Issues[0].Message; got != "至少需要 3 个字符" {
		t.Fatalf("message = %q", got)
	}
	if validation.Issues[0].Code != CodeTooSmall {
		t.Fatalf("code = %q", validation.Issues[0].Code)
	}
}

func TestWithLocaleOverridesSetLanguage(t *testing.T) {
	previous := Language()
	t.Cleanup(func() { SetLanguage(previous) })

	SetLanguage("zh-CN")
	ctx := WithLocale(context.Background(), "en")
	_, err := String().Email().ParseContext(ctx, "plain")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if got := validation.Issues[0].Message; got != "must be a valid email address" {
		t.Fatalf("message = %q", got)
	}
}

func TestLocalizeRewritesExistingError(t *testing.T) {
	_, err := String().Min(2).Parse("x")
	localized := Localize(err, "zh-CN")
	var validation *ValidationError
	if !errors.As(localized, &validation) {
		t.Fatal(localized)
	}
	if got := validation.Issues[0].Message; got != "至少需要 2 个字符" {
		t.Fatalf("message = %q", got)
	}
}

func TestMessageCatalogsShareKeys(t *testing.T) {
	t.Parallel()
	loadMessageCatalogs()
	if messageCatalogsErr != nil {
		t.Fatal(messageCatalogsErr)
	}
	en := messageCatalogs["en"]
	zh := messageCatalogs["zh-CN"]
	if len(en) == 0 || len(zh) == 0 {
		t.Fatalf("catalog sizes en=%d zh=%d", len(en), len(zh))
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Fatalf("zh-CN missing key %q", key)
		}
	}
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Fatalf("en missing key %q", key)
		}
	}
}

func TestEnglishMessagesComeFromCatalog(t *testing.T) {
	t.Parallel()
	_, err := String().Min(3).Parse("x")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	if got := validation.Issues[0].Message; got != "must contain at least 3 characters" {
		t.Fatalf("message = %q", got)
	}
}
