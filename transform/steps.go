package transform

import (
	"context"
	"strings"
	"unicode"
)

type step[T any] func(context.Context, T) (T, error)

func function[T any](fn func(T) (T, error)) step[T] {
	if fn == nil {
		panic("transform: nil function")
	}
	return func(_ context.Context, v T) (T, error) { return fn(v) }
}
func functionContext[T any](fn func(context.Context, T) (T, error)) step[T] {
	if fn == nil {
		panic("transform: nil function")
	}
	return step[T](fn)
}

type trimSide uint8

const (
	trimBoth trimSide = iota
	trimLeft
	trimRight
)

func trim(side trimSide, chars []string) step[string] {
	if len(chars) > 1 {
		panic("transform: Trim accepts at most one character set")
	}
	var cutset string
	if len(chars) == 1 {
		cutset = chars[0]
	}
	return function(func(v string) (string, error) {
		if len(chars) == 0 {
			switch side {
			case trimLeft:
				return strings.TrimLeftFunc(v, unicode.IsSpace), nil
			case trimRight:
				return strings.TrimRightFunc(v, unicode.IsSpace), nil
			}
			return strings.TrimSpace(v), nil
		}
		switch side {
		case trimLeft:
			return strings.TrimLeft(v, cutset), nil
		case trimRight:
			return strings.TrimRight(v, cutset), nil
		}
		return strings.Trim(v, cutset), nil
	})
}
func lower() step[string] {
	return function(func(v string) (string, error) { return strings.ToLower(v), nil })
}
func upper() step[string] {
	return function(func(v string) (string, error) { return strings.ToUpper(v), nil })
}
