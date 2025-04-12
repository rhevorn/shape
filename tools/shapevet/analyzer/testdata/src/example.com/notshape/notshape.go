package notshape

func Struct[T any]() int               { return 0 }
func BindJSON[T any](*T, []byte) error { return nil }
func New[T any](...any) int            { return 0 }
