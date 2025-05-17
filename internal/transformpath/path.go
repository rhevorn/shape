// Package transformpath carries collection paths across the public transform
// package boundary without exposing an intermediate error type to users.
package transformpath

type Segment struct {
	Key     any
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
	return prefix(err, Segment{Key: key})
}

func prefix(err error, segment Segment) error {
	if err == nil {
		return nil
	}
	// Each wrapper owns exactly one segment; the root assembles the full path.
	return &Error{Segments: []Segment{segment}, Err: err}
}
