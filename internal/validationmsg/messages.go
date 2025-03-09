// Package validationmsg owns built-in validation message templates shared by
// explicit validators and tag-driven schemas.
package validationmsg

import "fmt"

func Render(chinese bool, id, label string, expected any) string {
	if chinese {
		switch id {
		case "not_empty.string":
			return label + "不能为空"
		case "not_empty.collection":
			return label + "至少需要一项"
		case "not_empty.pointer", "not_null":
			return label + "不能为 null"
		case "too_small.string":
			return fmt.Sprintf("%s至少需要 %v 个字符", label, expected)
		case "too_big.string":
			return fmt.Sprintf("%s最多允许 %v 个字符", label, expected)
		case "string.len":
			return fmt.Sprintf("%s必须正好包含 %v 个字符", label, expected)
		case "too_small.collection":
			return fmt.Sprintf("%s至少需要 %v 项", label, expected)
		case "too_big.collection":
			return fmt.Sprintf("%s最多允许 %v 项", label, expected)
		case "collection.len":
			return fmt.Sprintf("%s必须正好包含 %v 项", label, expected)
		case "number.gt":
			return fmt.Sprintf("%s必须大于 %v", label, expected)
		case "number.gte":
			return fmt.Sprintf("%s必须大于等于 %v", label, expected)
		case "number.lt":
			return fmt.Sprintf("%s必须小于 %v", label, expected)
		case "number.lte":
			return fmt.Sprintf("%s必须小于等于 %v", label, expected)
		case "number.between":
			return label + "必须位于允许范围内"
		case "number.finite":
			return label + "必须是有限数值"
		case "invalid_enum":
			return label + "必须是允许值之一"
		case "pattern":
			return label + "格式不符合要求"
		case "startswith":
			return fmt.Sprintf("%s必须以 %q 开头", label, expected)
		case "endswith":
			return fmt.Sprintf("%s必须以 %q 结尾", label, expected)
		case "contains":
			return fmt.Sprintf("%s必须包含 %q", label, expected)
		case "email":
			return label + "必须是有效邮箱地址"
		case "url":
			return label + "必须是有效绝对 URL"
		case "uuid":
			return label + "必须是有效 UUID"
		case "ip":
			return label + "必须是有效 IP 地址"
		case "unique":
			return label + "不能包含重复项"
		case "unique_limit":
			return fmt.Sprintf("%s最多允许 %v 项进行深度去重检查", label, expected)
		case "too_many_issues":
			return fmt.Sprintf("校验错误过多，已在 %v 项后停止", expected)
		case "too_deep":
			return fmt.Sprintf("遍历深度不能超过 %v 层", expected)
		default:
			return label + "值无效"
		}
	}
	prefix := prefixLabel(label)
	switch id {
	case "not_empty.string":
		return prefix + "must not be empty"
	case "not_empty.collection":
		return prefix + "must contain at least one item"
	case "not_empty.pointer", "not_null":
		return prefix + "must not be null"
	case "too_small.string":
		return fmt.Sprintf("%smust contain at least %v characters", prefix, expected)
	case "too_big.string":
		return fmt.Sprintf("%smust contain at most %v characters", prefix, expected)
	case "string.len":
		return fmt.Sprintf("%smust contain exactly %v characters", prefix, expected)
	case "too_small.collection":
		return fmt.Sprintf("%smust contain at least %v items", prefix, expected)
	case "too_big.collection":
		return fmt.Sprintf("%smust contain at most %v items", prefix, expected)
	case "collection.len":
		return fmt.Sprintf("%smust contain exactly %v items", prefix, expected)
	case "number.gt":
		return fmt.Sprintf("%smust be greater than %v", prefix, expected)
	case "number.gte":
		return fmt.Sprintf("%smust be greater than or equal to %v", prefix, expected)
	case "number.lt":
		return fmt.Sprintf("%smust be less than %v", prefix, expected)
	case "number.lte":
		return fmt.Sprintf("%smust be less than or equal to %v", prefix, expected)
	case "number.between":
		return prefix + "must be within the allowed range"
	case "number.finite":
		return prefix + "must be a finite number"
	case "invalid_enum":
		return prefix + "must be one of the allowed values"
	case "pattern":
		return prefix + "must match the required pattern"
	case "startswith":
		return fmt.Sprintf("%smust start with %q", prefix, expected)
	case "endswith":
		return fmt.Sprintf("%smust end with %q", prefix, expected)
	case "contains":
		return fmt.Sprintf("%smust contain %q", prefix, expected)
	case "email":
		return prefix + "must be a valid email address"
	case "url":
		return prefix + "must be a valid absolute URL"
	case "uuid":
		return prefix + "must be a valid UUID"
	case "ip":
		return prefix + "must be a valid IP address"
	case "unique":
		return prefix + "must contain unique items"
	case "unique_limit":
		return fmt.Sprintf("%sdeep uniqueness checks support at most %v items", prefix, expected)
	case "too_many_issues":
		return fmt.Sprintf("validation stopped after %v issues", expected)
	case "too_deep":
		return fmt.Sprintf("maximum traversal depth of %v exceeded", expected)
	default:
		return prefix + "has an invalid value"
	}
}

func prefixLabel(value string) string {
	if value == "" {
		return ""
	}
	return value + " "
}
