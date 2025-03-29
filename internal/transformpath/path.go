// Package transformpath carries collection paths across the public transform
// package boundary without exposing an intermediate error type to users.
package transformpath

import (
	"errors"
	"fmt"
)

type Segment struct {
	Key     string
	Index   int
	IsIndex bool
}

type Error struct {
	Segments []Segment
	Err      error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return "transform failed"
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Index(err error, index int) error {
	return prefix(err, Segment{Index: index, IsIndex: true})
}

func Key(err error, key any) error {
	return prefix(err, Segment{Key: fmt.Sprint(key)})
}

func prefix(err error, segment Segment) error {
	if err == nil {
		return nil
	}
	// errors.As, not a bare type assertion: user code may wrap the error with
	// %w, and the inner segments must survive that. err is kept whole so the
	// user's wrapper still renders its own message.
	var nested *Error
	if errors.As(err, &nested) && nested != nil {
		segments := make([]Segment, 0, len(nested.Segments)+1)
		segments = append(segments, segment)
		segments = append(segments, nested.Segments...)
		return &Error{Segments: segments, Err: err}
	}
	return &Error{Segments: []Segment{segment}, Err: err}
}
