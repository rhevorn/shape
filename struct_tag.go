package shape

import (
	"fmt"
	"github.com/rhevorn/shape/internal/taglang"
	"strconv"
)

type tagOption struct {
	name, value string
	has         bool
}

func parseShapeTag(text string) ([]tagOption, error) {
	items, err := taglang.Parse(text)
	if err != nil {
		return nil, fmt.Errorf("shape: %w", err)
	}
	out := make([]tagOption, len(items))
	for i, item := range items {
		out[i] = tagOption{item.Name, item.Value, item.HasValue}
	}
	return out, nil
}
func tagCount(text string) (int, error) {
	if text == "" {
		return 0, fmt.Errorf("empty count")
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid non-negative count %q", text)
		}
	}
	return strconv.Atoi(text)
}
