// Package types provides data types with explicit JSON representations.
package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

// Duration is a signed nanosecond duration encoded as a JSON string ("30s").
type Duration time.Duration

// String formats d with time.Duration syntax.
func (d Duration) String() string { return time.Duration(d).String() }

// MarshalJSON encodes d as a duration string.
func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON accepts duration strings or null (zero). Errors do not modify d.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return errors.New("shape: nil Duration receiver")
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*d = 0
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	value, err := time.ParseDuration(text)
	if err != nil {
		return err
	}
	*d = Duration(value)
	return nil
}
