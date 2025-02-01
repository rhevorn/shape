package shape

import (
	"context"
	"time"
)

type apiContractObject struct {
	Name string
}

type apiContractTuple struct {
	Name string
	Age  int
}

type apiContractDefaults struct {
	Labels map[string]string
}

var (
	_ Schema[string]            = String()
	_ Schema[int]               = Int()
	_ Schema[int64]             = Int64()
	_ Schema[float64]           = Float64()
	_ Schema[bool]              = Bool()
	_ Schema[time.Time]         = Time()
	_ Schema[time.Duration]     = Duration()
	_ Schema[uint16]            = Number[uint16]()
	_ Schema[[]string]          = Slice(String())
	_ Schema[map[string]int]    = Map(Int())
	_ Schema[map[string]int]    = Record(String(), Int())
	_ Schema[*string]           = Nullable(String())
	_ Schema[string]            = Enum("open", "closed")
	_ Schema[string]            = Literal("fixed")
	_ Schema[string]            = Union[string](String(), UUID())
	_ Schema[string]            = OneOf[string](String().Email(), UUID())
	_ Schema[int]               = Transform(String(), func(string) (int, error) { return 0, nil })
	_ Schema[string]            = Refine(String(), func(string) error { return nil })
	_ Schema[string]            = RefineContext(String(), func(_ context.Context, _ string) error { return nil })
	_ Schema[string]            = Annotate(String())
	_ Schema[string]            = Label("name", String())
	_ ObjectField[testUser]     = Fields[testUser]().Str("name").Set(func(*testUser, string) {})
	_ ObjectField[testUser]     = Fields[testUser]().Int("age").Set(func(*testUser, int) {})
	_ Schema[apiContractObject] = Object[apiContractObject](
		Field("name", String(), func(value *apiContractObject, name string) { value.Name = name }),
	)
	_ Schema[apiContractTuple] = Tuple[apiContractTuple](
		TupleItem(String(), func(value *apiContractTuple, name string) { value.Name = name }),
		TupleItem(Int(), func(value *apiContractTuple, age int) { value.Age = age }),
	)
	_ Schema[apiContractObject] = Lazy("APIContractObject", func() Schema[apiContractObject] {
		return Object[apiContractObject](
			Field("name", String(), func(value *apiContractObject, name string) { value.Name = name }),
		)
	}).MaxDepth(DefaultMaxRecursiveDepth)
	_ Schema[apiContractDefaults] = Object[apiContractDefaults](
		Field("labels", Map(String()), func(value *apiContractDefaults, labels map[string]string) {
			value.Labels = labels
		}).DefaultFunc(func() map[string]string { return make(map[string]string) }),
	)
)
