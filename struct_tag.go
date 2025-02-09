package shape

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tagOption struct {
	name  string
	value string
	has   bool // true when name=value form
}

// parseShapeTag parses a shape struct tag into ordered options.
// Forms: name | name=value | name='value' | name="value"
// Options are comma-separated. Commas inside quotes do not split.
func parseShapeTag(tag string) ([]tagOption, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, nil
	}
	var options []tagOption
	for len(tag) > 0 {
		tag = strings.TrimLeftFunc(tag, unicode.IsSpace)
		if tag == "" {
			break
		}
		name, rest, err := readTagName(tag)
		if err != nil {
			return nil, err
		}
		tag = rest
		opt := tagOption{name: strings.ToLower(name)}
		if strings.HasPrefix(tag, "=") {
			tag = tag[1:]
			value, next, err := readTagValue(tag)
			if err != nil {
				return nil, err
			}
			opt.value = value
			opt.has = true
			tag = next
		}
		options = append(options, opt)
		tag = strings.TrimLeftFunc(tag, unicode.IsSpace)
		if tag == "" {
			break
		}
		if tag[0] != ',' {
			return nil, fmt.Errorf("shape: expected ',' in tag after %q", opt.name)
		}
		tag = tag[1:]
	}
	return options, nil
}

func readTagName(tag string) (string, string, error) {
	if tag == "" {
		return "", "", fmt.Errorf("shape: empty tag option name")
	}
	i := 0
	for i < len(tag) {
		r, size := utf8.DecodeRuneInString(tag[i:])
		if r == '=' || r == ',' || unicode.IsSpace(r) {
			break
		}
		i += size
	}
	if i == 0 {
		return "", "", fmt.Errorf("shape: empty tag option name")
	}
	return tag[:i], tag[i:], nil
}

func readTagValue(tag string) (string, string, error) {
	tag = strings.TrimLeftFunc(tag, unicode.IsSpace)
	if tag == "" {
		return "", "", nil
	}
	switch tag[0] {
	case '\'', '"':
		quote := tag[0]
		var b strings.Builder
		i := 1
		for i < len(tag) {
			c := tag[i]
			if c == '\\' && i+1 < len(tag) {
				b.WriteByte(tag[i+1])
				i += 2
				continue
			}
			if c == quote {
				return b.String(), tag[i+1:], nil
			}
			b.WriteByte(c)
			i++
		}
		return "", "", fmt.Errorf("shape: unterminated quoted tag value")
	default:
		i := 0
		for i < len(tag) && tag[i] != ',' {
			i++
		}
		return strings.TrimSpace(tag[:i]), tag[i:], nil
	}
}

func tagInt(opt tagOption) (int, error) {
	if !opt.has {
		return 0, fmt.Errorf("shape: tag %q requires a value", opt.name)
	}
	n, err := strconv.Atoi(strings.TrimSpace(opt.value))
	if err != nil {
		return 0, fmt.Errorf("shape: tag %q value %q is not an int", opt.name, opt.value)
	}
	return n, nil
}

func tagInt64(opt tagOption) (int64, error) {
	if !opt.has {
		return 0, fmt.Errorf("shape: tag %q requires a value", opt.name)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(opt.value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("shape: tag %q value %q is not an int64", opt.name, opt.value)
	}
	return n, nil
}

func tagFloat64(opt tagOption) (float64, error) {
	if !opt.has {
		return 0, fmt.Errorf("shape: tag %q requires a value", opt.name)
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(opt.value), 64)
	if err != nil {
		return 0, fmt.Errorf("shape: tag %q value %q is not a float", opt.name, opt.value)
	}
	return n, nil
}

func tagRequiredString(opt tagOption) (string, error) {
	if !opt.has {
		return "", fmt.Errorf("shape: tag %q requires a value", opt.name)
	}
	return opt.value, nil
}

func splitOneOf(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '|' || r == ',' || unicode.IsSpace(r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
