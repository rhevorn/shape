package validationmsg

import (
	"strings"
	"testing"

	"github.com/rhevorn/shape/internal/validationlocale"
)

func TestEveryMessageHasBothLanguages(t *testing.T) {
	for id := range messages {
		english := Render(validationlocale.English, id, "field", 3)
		chinese := Render(validationlocale.SimplifiedChinese, id, "field", 3)
		if strings.TrimSpace(english) == "" || strings.TrimSpace(chinese) == "" || english == chinese {
			t.Fatalf("message %q: english=%q chinese=%q", id, english, chinese)
		}
	}
}

func TestUnknownMessageUsesLocalizedFallback(t *testing.T) {
	if got := Render(validationlocale.English, "missing", "field", nil); got != "field has an invalid value" {
		t.Fatalf("English fallback = %q", got)
	}
	if got := Render(validationlocale.SimplifiedChinese, "missing", "field", nil); got != "field值无效" {
		t.Fatalf("Chinese fallback = %q", got)
	}
}
