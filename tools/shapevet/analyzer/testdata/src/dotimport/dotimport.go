package dotimport

import . "github.com/rhevorn/shape"

type Bad struct {
	Value any
}

func build() {
	_ = Struct[Bad]()                        // want "unsupported field type"
	_ = New[Bad](Field("Missing", String())) // want "target has no direct field Missing"
}
