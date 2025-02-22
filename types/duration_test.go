package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDurationJSON(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  Duration
	}{{"\"30s\"", Duration(30 * time.Second)}, {"\"-500ms\"", Duration(-500 * time.Millisecond)}, {"null", 0}, {"\"0\"", 0}} {
		d := Duration(time.Hour)
		if err := json.Unmarshal([]byte(tc.input), &d); err != nil || d != tc.want {
			t.Fatalf("%s: %v %v", tc.input, d, err)
		}
		encoded, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		var round Duration
		if err = json.Unmarshal(encoded, &round); err != nil || round != d {
			t.Fatal(string(encoded), err)
		}
	}
	for _, input := range []string{"30", "true", "\"\"", "\"abc\"", "\"999999999999999999999h\"", "{}"} {
		d := Duration(time.Second)
		if err := json.Unmarshal([]byte(input), &d); err == nil || d != Duration(time.Second) {
			t.Fatalf("%s: %v", input, d)
		}
	}
	var p *Duration
	if err := json.Unmarshal([]byte("null"), &p); err != nil || p != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte("\"1s\""), &p); err != nil || p == nil || *p != Duration(time.Second) {
		t.Fatal(err)
	}
}
