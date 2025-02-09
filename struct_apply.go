package shape

import (
	"fmt"
	"regexp"
	"strconv"
)

func applyStringTags(schema StringSchema, options []tagOption) (StringSchema, error) {
	for _, opt := range options {
		switch opt.name {
		case "trim":
			if opt.has {
				return schema, fmt.Errorf("tag trim takes no value")
			}
			schema = schema.Trim()
		case "tolower":
			if opt.has {
				return schema, fmt.Errorf("tag tolower takes no value")
			}
			schema = schema.ToLower()
		case "toupper":
			if opt.has {
				return schema, fmt.Errorf("tag toupper takes no value")
			}
			schema = schema.ToUpper()
		case "nonempty":
			if opt.has {
				return schema, fmt.Errorf("tag nonempty takes no value")
			}
			schema = schema.NonEmpty()
		case "min":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Min(n)
		case "max":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Max(n)
		case "len":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Len(n)
		case "pattern":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			re, err := regexp.Compile(value)
			if err != nil {
				return schema, fmt.Errorf("tag pattern: %w", err)
			}
			schema = schema.Pattern(re)
		case "startswith":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.StartsWith(value)
		case "endswith":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.EndsWith(value)
		case "contains":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Contains(value)
		case "email":
			if opt.has {
				return schema, fmt.Errorf("tag email takes no value")
			}
			schema = schema.Email()
		case "url":
			if opt.has {
				return schema, fmt.Errorf("tag url takes no value")
			}
			schema = schema.URL()
		case "uuid":
			if opt.has {
				return schema, fmt.Errorf("tag uuid takes no value")
			}
			schema = schema.UUID()
		case "ip":
			if opt.has {
				return schema, fmt.Errorf("tag ip takes no value")
			}
			schema = schema.IP()
		default:
			return schema, fmt.Errorf("unsupported string tag %q", opt.name)
		}
	}
	return schema, nil
}

func applyIntTags(schema IntSchema, options []tagOption) (IntSchema, error) {
	for _, opt := range options {
		switch opt.name {
		case "min":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Min(n)
		case "max":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Max(n)
		case "gt":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Gt(n)
		case "gte":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Gte(n)
		case "lt":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Lt(n)
		case "lte":
			n, err := tagInt(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Lte(n)
		case "positive":
			if opt.has {
				return schema, fmt.Errorf("tag positive takes no value")
			}
			schema = schema.Positive()
		case "negative":
			if opt.has {
				return schema, fmt.Errorf("tag negative takes no value")
			}
			schema = schema.Negative()
		case "nonnegative":
			if opt.has {
				return schema, fmt.Errorf("tag nonnegative takes no value")
			}
			schema = schema.NonNegative()
		case "oneof":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			parts := splitOneOf(value)
			if len(parts) == 0 {
				return schema, fmt.Errorf("tag oneof requires at least one value")
			}
			values := make([]int, len(parts))
			for i, part := range parts {
				n, err := strconv.Atoi(part)
				if err != nil {
					return schema, fmt.Errorf("tag oneof value %q is not an int", part)
				}
				values[i] = n
			}
			schema = schema.OneOf(values...)
		default:
			return schema, fmt.Errorf("unsupported int tag %q", opt.name)
		}
	}
	return schema, nil
}

func applyInt64Tags(schema Int64Schema, options []tagOption) (Int64Schema, error) {
	for _, opt := range options {
		switch opt.name {
		case "min":
			n, err := tagInt64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Min(n)
		case "max":
			n, err := tagInt64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Max(n)
		case "gt":
			n, err := tagInt64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Gt(n)
		case "gte":
			n, err := tagInt64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Gte(n)
		case "lt":
			n, err := tagInt64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Lt(n)
		case "lte":
			n, err := tagInt64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Lte(n)
		case "positive":
			if opt.has {
				return schema, fmt.Errorf("tag positive takes no value")
			}
			schema = schema.Positive()
		case "negative":
			if opt.has {
				return schema, fmt.Errorf("tag negative takes no value")
			}
			schema = schema.Negative()
		case "nonnegative":
			if opt.has {
				return schema, fmt.Errorf("tag nonnegative takes no value")
			}
			schema = schema.NonNegative()
		case "oneof":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			parts := splitOneOf(value)
			if len(parts) == 0 {
				return schema, fmt.Errorf("tag oneof requires at least one value")
			}
			values := make([]int64, len(parts))
			for i, part := range parts {
				n, err := strconv.ParseInt(part, 10, 64)
				if err != nil {
					return schema, fmt.Errorf("tag oneof value %q is not an int64", part)
				}
				values[i] = n
			}
			schema = schema.OneOf(values...)
		default:
			return schema, fmt.Errorf("unsupported int64 tag %q", opt.name)
		}
	}
	return schema, nil
}

func applyFloat64Tags(schema Float64Schema, options []tagOption) (Float64Schema, error) {
	for _, opt := range options {
		switch opt.name {
		case "min":
			n, err := tagFloat64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Min(n)
		case "max":
			n, err := tagFloat64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Max(n)
		case "gt":
			n, err := tagFloat64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Gt(n)
		case "gte":
			n, err := tagFloat64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Gte(n)
		case "lt":
			n, err := tagFloat64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Lt(n)
		case "lte":
			n, err := tagFloat64(opt)
			if err != nil {
				return schema, err
			}
			schema = schema.Lte(n)
		case "positive":
			if opt.has {
				return schema, fmt.Errorf("tag positive takes no value")
			}
			schema = schema.Positive()
		case "negative":
			if opt.has {
				return schema, fmt.Errorf("tag negative takes no value")
			}
			schema = schema.Negative()
		case "nonnegative":
			if opt.has {
				return schema, fmt.Errorf("tag nonnegative takes no value")
			}
			schema = schema.NonNegative()
		case "oneof":
			value, err := tagRequiredString(opt)
			if err != nil {
				return schema, err
			}
			parts := splitOneOf(value)
			if len(parts) == 0 {
				return schema, fmt.Errorf("tag oneof requires at least one value")
			}
			values := make([]float64, len(parts))
			for i, part := range parts {
				n, err := strconv.ParseFloat(part, 64)
				if err != nil {
					return schema, fmt.Errorf("tag oneof value %q is not a float", part)
				}
				values[i] = n
			}
			schema = schema.OneOf(values...)
		default:
			return schema, fmt.Errorf("unsupported float64 tag %q", opt.name)
		}
	}
	return schema, nil
}

func applyDurationTags(schema DurationSchema, options []tagOption) (DurationSchema, error) {
	for _, opt := range options {
		switch opt.name {
		case "coerce":
			if opt.has {
				return schema, fmt.Errorf("tag coerce takes no value")
			}
			schema = CoerceDuration()
		default:
			return schema, fmt.Errorf("unsupported duration tag %q", opt.name)
		}
	}
	return schema, nil
}

func applyTimeTags(schema TimeSchema, options []tagOption) (TimeSchema, error) {
	for _, opt := range options {
		switch opt.name {
		case "coerce":
			if opt.has {
				return schema, fmt.Errorf("tag coerce takes no value")
			}
			schema = CoerceTime()
		default:
			return schema, fmt.Errorf("unsupported time tag %q", opt.name)
		}
	}
	return schema, nil
}
