package validate

import "fmt"

func localizeIssue(v Issue, lang Language) Issue {
	if v.MessageID == "" && v.Message != "" {
		return v
	}
	zh := lang == SimplifiedChinese
	label := v.Label
	if zh {
		switch v.MessageID {
		case "not_empty.string":
			v.Message = label + "不能为空"
		case "not_empty.collection":
			v.Message = label + "至少需要一项"
		case "not_empty.pointer", "not_null":
			v.Message = label + "不能为 null"
		case "too_small.string":
			v.Message = fmt.Sprintf("%s至少需要 %v 个字符", label, v.Expected)
		case "too_big.string":
			v.Message = fmt.Sprintf("%s最多允许 %v 个字符", label, v.Expected)
		case "string.len":
			v.Message = fmt.Sprintf("%s必须正好包含 %v 个字符", label, v.Expected)
		case "too_small.collection":
			v.Message = fmt.Sprintf("%s至少需要 %v 项", label, v.Expected)
		case "too_big.collection":
			v.Message = fmt.Sprintf("%s最多允许 %v 项", label, v.Expected)
		case "collection.len":
			v.Message = fmt.Sprintf("%s必须正好包含 %v 项", label, v.Expected)
		case "number.gt":
			v.Message = fmt.Sprintf("%s必须大于 %v", label, v.Expected)
		case "number.gte":
			v.Message = fmt.Sprintf("%s必须大于等于 %v", label, v.Expected)
		case "number.lt":
			v.Message = fmt.Sprintf("%s必须小于 %v", label, v.Expected)
		case "number.lte":
			v.Message = fmt.Sprintf("%s必须小于等于 %v", label, v.Expected)
		case "number.between":
			v.Message = label + "必须位于允许范围内"
		case "number.finite":
			v.Message = label + "必须是有限数值"
		case "invalid_enum":
			v.Message = label + "必须是允许值之一"
		case "pattern":
			v.Message = label + "格式不符合要求"
		case "startswith":
			v.Message = fmt.Sprintf("%s必须以 %q 开头", label, v.Expected)
		case "endswith":
			v.Message = fmt.Sprintf("%s必须以 %q 结尾", label, v.Expected)
		case "contains":
			v.Message = fmt.Sprintf("%s必须包含 %q", label, v.Expected)
		case "email":
			v.Message = label + "必须是有效邮箱地址"
		case "url":
			v.Message = label + "必须是有效绝对 URL"
		case "uuid":
			v.Message = label + "必须是有效 UUID"
		case "ip":
			v.Message = label + "必须是有效 IP 地址"
		case "unique":
			v.Message = label + "不能包含重复项"
		case "unique_limit":
			v.Message = fmt.Sprintf("%s最多允许 %v 项进行深度去重检查", label, v.Expected)
		case "too_many_issues":
			v.Message = fmt.Sprintf("校验错误过多，已在 %v 项后停止", v.Expected)
		case "too_deep":
			v.Message = fmt.Sprintf("遍历深度不能超过 %v 层", v.Expected)
		default:
			v.Message = label + "值无效"
		}
		return v
	}
	switch v.MessageID {
	case "not_empty.string":
		v.Message = prefixLabel(label) + "must not be empty"
	case "not_empty.collection":
		v.Message = prefixLabel(label) + "must contain at least one item"
	case "not_empty.pointer", "not_null":
		v.Message = prefixLabel(label) + "must not be null"
	case "too_small.string":
		v.Message = fmt.Sprintf("%smust contain at least %v characters", prefixLabel(label), v.Expected)
	case "too_big.string":
		v.Message = fmt.Sprintf("%smust contain at most %v characters", prefixLabel(label), v.Expected)
	case "string.len":
		v.Message = fmt.Sprintf("%smust contain exactly %v characters", prefixLabel(label), v.Expected)
	case "too_small.collection":
		v.Message = fmt.Sprintf("%smust contain at least %v items", prefixLabel(label), v.Expected)
	case "too_big.collection":
		v.Message = fmt.Sprintf("%smust contain at most %v items", prefixLabel(label), v.Expected)
	case "collection.len":
		v.Message = fmt.Sprintf("%smust contain exactly %v items", prefixLabel(label), v.Expected)
	case "number.gt":
		v.Message = fmt.Sprintf("%smust be greater than %v", prefixLabel(label), v.Expected)
	case "number.gte":
		v.Message = fmt.Sprintf("%smust be greater than or equal to %v", prefixLabel(label), v.Expected)
	case "number.lt":
		v.Message = fmt.Sprintf("%smust be less than %v", prefixLabel(label), v.Expected)
	case "number.lte":
		v.Message = fmt.Sprintf("%smust be less than or equal to %v", prefixLabel(label), v.Expected)
	case "number.between":
		v.Message = prefixLabel(label) + "must be within the allowed range"
	case "number.finite":
		v.Message = prefixLabel(label) + "must be a finite number"
	case "invalid_enum":
		v.Message = prefixLabel(label) + "must be one of the allowed values"
	case "pattern":
		v.Message = prefixLabel(label) + "must match the required pattern"
	case "startswith":
		v.Message = fmt.Sprintf("%smust start with %q", prefixLabel(label), v.Expected)
	case "endswith":
		v.Message = fmt.Sprintf("%smust end with %q", prefixLabel(label), v.Expected)
	case "contains":
		v.Message = fmt.Sprintf("%smust contain %q", prefixLabel(label), v.Expected)
	case "email":
		v.Message = prefixLabel(label) + "must be a valid email address"
	case "url":
		v.Message = prefixLabel(label) + "must be a valid absolute URL"
	case "uuid":
		v.Message = prefixLabel(label) + "must be a valid UUID"
	case "ip":
		v.Message = prefixLabel(label) + "must be a valid IP address"
	case "unique":
		v.Message = prefixLabel(label) + "must contain unique items"
	case "unique_limit":
		v.Message = fmt.Sprintf("%sdeep uniqueness checks support at most %v items", prefixLabel(label), v.Expected)
	case "too_many_issues":
		v.Message = fmt.Sprintf("validation stopped after %v issues", v.Expected)
	case "too_deep":
		v.Message = fmt.Sprintf("maximum traversal depth of %v exceeded", v.Expected)
	default:
		v.Message = prefixLabel(label) + "has an invalid value"
	}
	return v
}

func prefixLabel(v string) string {
	if v == "" {
		return ""
	}
	return v + " "
}
