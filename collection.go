package goshape

import "fmt"

type lengthRule func(int) *Issue

func minLengthRule(kind string, minimum int) lengthRule {
	requireNonNegative(kind+".Min", minimum)
	return func(length int) *Issue {
		if length >= minimum {
			return nil
		}
		return &Issue{
			Code:     CodeTooSmall,
			Message:  fmt.Sprintf("must contain at least %d items", minimum),
			Expected: minimum,
			Received: length,
		}
	}
}

func maxLengthRule(kind string, maximum int) lengthRule {
	requireNonNegative(kind+".Max", maximum)
	return func(length int) *Issue {
		if length <= maximum {
			return nil
		}
		return &Issue{
			Code:     CodeTooBig,
			Message:  fmt.Sprintf("must contain at most %d items", maximum),
			Expected: maximum,
			Received: length,
		}
	}
}
