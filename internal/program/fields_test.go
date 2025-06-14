package program

import (
	"reflect"
	"testing"
)

func TestDirectJSONFieldReachability(t *testing.T) {
	stringType := reflect.TypeFor[string]()
	tests := []struct {
		name     string
		fields   []reflect.StructField
		bound    string
		rejected bool
	}{
		{"unbound duplicate", []reflect.StructField{{Name: "A", Type: stringType, Tag: `json:"name"`}, {Name: "B", Type: stringType, Tag: `json:"name"`}}, "A", true},
		{"tag shadows bound field", []reflect.StructField{{Name: "A", Type: stringType}, {Name: "B", Type: stringType, Tag: `json:"A"`}}, "A", true},
		{"tagged winner", []reflect.StructField{{Name: "A", Type: stringType}, {Name: "B", Type: stringType, Tag: `json:"A"`}}, "B", false},
		{"ignored sibling", []reflect.StructField{{Name: "A", Type: stringType}, {Name: "B", Type: stringType, Tag: `json:"-"`}}, "A", false},
		{"invalid tag falls back to Go name", []reflect.StructField{{Name: "A", Type: stringType, Tag: `json:"bad\\name"`}, {Name: "B", Type: stringType, Tag: `json:"A"`}}, "A", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if got := recover() != nil; got != tc.rejected {
					t.Errorf("construction panic=%v want=%v", got, tc.rejected)
				}
			}()
			Compile(reflect.StructOf(tc.fields), []Definition{{Name: tc.bound, Type: stringType}})
		})
	}
}
