// Package taglang parses Shape's shared struct-tag language.
package taglang

import (
	"fmt"
	"strings"
	"unicode"
)

type Item struct {
	Name, Value string
	HasValue    bool
}

func Parse(text string) ([]Item, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	var result []Item
	for {
		end := tagNameEnd(text)
		name := text[:end]
		if name == "" {
			return nil, fmt.Errorf("empty tag name")
		}
		for _, c := range name {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
				return nil, fmt.Errorf("invalid tag name %q", name)
			}
		}
		text = strings.TrimLeftFunc(text[end:], unicode.IsSpace)
		item := Item{Name: strings.ToLower(name)}
		if strings.HasPrefix(text, "=") {
			item.HasValue = true
			text = strings.TrimLeftFunc(text[1:], unicode.IsSpace)
			if strings.HasPrefix(text, "'") {
				text = text[1:]
				var b strings.Builder
				closed := false
				for len(text) > 0 {
					c := text[0]
					text = text[1:]
					if c == '\'' {
						closed = true
						break
					}
					if c == '\\' {
						if text == "" {
							return nil, fmt.Errorf("incomplete tag escape")
						}
						c = text[0]
						text = text[1:]
						switch c {
						case '\\', '\'':
						case 'n':
							c = '\n'
						case 'r':
							c = '\r'
						case 't':
							c = '\t'
						default:
							return nil, fmt.Errorf("invalid escape %q", c)
						}
					}
					b.WriteByte(c)
				}
				if !closed {
					return nil, fmt.Errorf("unterminated tag quote")
				}
				item.Value = b.String()
			} else {
				end = strings.IndexByte(text, ',')
				if end < 0 {
					end = len(text)
				}
				item.Value = strings.TrimSpace(text[:end])
				text = text[end:]
				if item.Value == "" || strings.ContainsAny(item.Value, "'\"") {
					return nil, fmt.Errorf("empty or invalid unquoted tag value")
				}
			}
		}
		result = append(result, item)
		text = strings.TrimSpace(text)
		if text == "" {
			return result, nil
		}
		if text[0] != ',' {
			return nil, fmt.Errorf("expected comma after %s", item.Name)
		}
		text = strings.TrimSpace(text[1:])
		if text == "" {
			return nil, fmt.Errorf("trailing comma")
		}
	}
}

func tagNameEnd(text string) int {
	for index, value := range text {
		if value == '=' || value == ',' || unicode.IsSpace(value) {
			return index
		}
	}
	return len(text)
}
