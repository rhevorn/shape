// Package jsonfields shares JSON field naming and direct-field dominance rules.
package jsonfields

import (
	"strings"
	"unicode"
)

// Name follows encoding/json's validation of an explicit field name.
func Name(goName, tag string) (name string, tagged bool) {
	name = strings.Split(tag, ",")[0]
	if name == "" {
		return goName, false
	}
	for _, r := range name {
		if !strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", r) && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return goName, false
		}
	}
	return name, true
}

type Candidate struct {
	Index  int
	Name   string
	Tagged bool
}

// Resolve chooses the direct field for each JSON name. One explicitly tagged
// field wins over untagged fields; ties are ambiguous and resolve to -1.
func Resolve(fields []Candidate) map[string]int {
	winners := make(map[string]int, len(fields))
	tagged := make(map[string]bool, len(fields))
	for _, field := range fields {
		_, exists := winners[field.Name]
		switch {
		case !exists || field.Tagged && !tagged[field.Name]:
			winners[field.Name], tagged[field.Name] = field.Index, field.Tagged
		case field.Tagged == tagged[field.Name]:
			winners[field.Name] = -1
		}
	}
	return winners
}
