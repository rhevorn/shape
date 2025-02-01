package shape

type lengthRule func(int) *Issue

func minLengthRule(kind string, minimum int) lengthRule {
	requireNonNegative(kind+".Min", minimum)
	return func(length int) *Issue {
		if length >= minimum {
			return nil
		}
		issue := keyedIssue(CodeTooSmall, "too_small.collection", minimum, length)
		return &issue
	}
}

func maxLengthRule(kind string, maximum int) lengthRule {
	requireNonNegative(kind+".Max", maximum)
	return func(length int) *Issue {
		if length <= maximum {
			return nil
		}
		issue := keyedIssue(CodeTooBig, "too_big.collection", maximum, length)
		return &issue
	}
}
