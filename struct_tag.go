package shape

import (
	"fmt"
	"strconv"

	"github.com/rhevorn/shape/internal/taglang"
)

func parseShapeTag(text string) ([]taglang.Item, error) {
	items, err := taglang.Parse(text)
	if err != nil {
		return nil, fmt.Errorf("shape: %w", err)
	}
	return items, nil
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
