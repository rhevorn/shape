package validate

import "github.com/rhevorn/shape/internal/validationmsg"

func renderIssue(value Issue, lang Language) Issue {
	if value.messageID != "" {
		value.Message = validationmsg.Render(lang, value.messageID, value.Label, value.Expected)
	}
	return value
}
