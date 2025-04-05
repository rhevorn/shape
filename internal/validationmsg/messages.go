// Package validationmsg owns built-in validation message templates shared by
// explicit validators and tag-driven schemas.
package validationmsg

import (
	"fmt"

	"github.com/rhevorn/shape/internal/validationlocale"
)

type renderer func(label string, expected any) string

type message struct {
	english renderer
	chinese renderer
}

func literal(english, chinese string) message {
	return message{
		english: func(label string, _ any) string { return prefixLabel(label) + english },
		chinese: func(label string, _ any) string { return label + chinese },
	}
}

func expected(english, chinese string) message {
	return message{
		english: func(label string, value any) string { return prefixLabel(label) + fmt.Sprintf(english, value) },
		chinese: func(label string, value any) string { return label + fmt.Sprintf(chinese, value) },
	}
}

var messages = map[string]message{
	"not_empty.string":     literal("must not be empty", "不能为空"),
	"not_empty.collection": literal("must contain at least one item", "至少需要一项"),
	"not_empty.pointer":    literal("must not be null", "不能为 null"),
	"not_null":             literal("must not be null", "不能为 null"),
	"too_small.string":     expected("must contain at least %v characters", "至少需要 %v 个字符"),
	"too_big.string":       expected("must contain at most %v characters", "最多允许 %v 个字符"),
	"string.len":           expected("must contain exactly %v characters", "必须正好包含 %v 个字符"),
	"too_small.collection": expected("must contain at least %v items", "至少需要 %v 项"),
	"too_big.collection":   expected("must contain at most %v items", "最多允许 %v 项"),
	"collection.len":       expected("must contain exactly %v items", "必须正好包含 %v 项"),
	"number.gt":            expected("must be greater than %v", "必须大于 %v"),
	"number.gte":           expected("must be greater than or equal to %v", "必须大于等于 %v"),
	"number.lt":            expected("must be less than %v", "必须小于 %v"),
	"number.lte":           expected("must be less than or equal to %v", "必须小于等于 %v"),
	"number.between":       literal("must be within the allowed range", "必须位于允许范围内"),
	"number.finite":        literal("must be a finite number", "必须是有限数值"),
	"invalid_enum":         literal("must be one of the allowed values", "必须是允许值之一"),
	"pattern":              literal("must match the required pattern", "格式不符合要求"),
	"startswith":           expected("must start with %q", "必须以 %q 开头"),
	"endswith":             expected("must end with %q", "必须以 %q 结尾"),
	"contains":             expected("must contain %q", "必须包含 %q"),
	"email":                literal("must be a valid email address", "必须是有效邮箱地址"),
	"url":                  literal("must be a valid absolute URL", "必须是有效绝对 URL"),
	"uuid":                 literal("must be a valid UUID", "必须是有效 UUID"),
	"ip":                   literal("must be a valid IP address", "必须是有效 IP 地址"),
	"unique":               literal("must contain unique items", "不能包含重复项"),
	"unique_limit":         expected("deep uniqueness checks support at most %v items", "最多允许 %v 项进行深度去重检查"),
	"too_many_issues": {
		english: func(_ string, value any) string { return fmt.Sprintf("validation stopped after %v issues", value) },
		chinese: func(_ string, value any) string {
			return fmt.Sprintf("校验错误过多，已在 %v 项后停止", value)
		},
	},
	"too_deep": {
		english: func(_ string, value any) string { return fmt.Sprintf("maximum traversal depth of %v exceeded", value) },
		chinese: func(_ string, value any) string { return fmt.Sprintf("遍历深度不能超过 %v 层", value) },
	},
}

// Render formats one built-in validation message in language.
func Render(language validationlocale.Language, id, label string, expected any) string {
	template, ok := messages[id]
	if !ok {
		template = literal("has an invalid value", "值无效")
	}
	switch language {
	case validationlocale.English:
		return template.english(label, expected)
	case validationlocale.SimplifiedChinese:
		return template.chinese(label, expected)
	default:
		panic("validationmsg: unsupported language")
	}
}

func prefixLabel(value string) string {
	if value == "" {
		return ""
	}
	return value + " "
}
