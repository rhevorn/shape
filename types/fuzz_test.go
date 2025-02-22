package types_test

import (
	"encoding/json"
	"testing"

	"github.com/rhevorn/shape/types"
)

func FuzzDurationJSON(f *testing.F) {
	f.Add([]byte(`"30s"`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"bad"`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var value types.Duration
		if json.Unmarshal(data, &value) != nil {
			return
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var round types.Duration
		if err := json.Unmarshal(encoded, &round); err != nil || round != value {
			t.Fatalf("round=%v value=%v err=%v", round, value, err)
		}
	})
}
