package falsepositive

import shape "example.com/notshape"

type Unsupported struct {
	Value any
}

func build(data []byte) {
	_ = shape.Struct[Unsupported]()
	var value Unsupported
	_ = shape.BindJSON(&value, data)
	_ = shape.New[Unsupported]("Missing")
}
