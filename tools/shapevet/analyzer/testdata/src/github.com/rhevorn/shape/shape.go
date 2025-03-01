package shape

func Struct[T any]() int { return 0 }

func BindJSON[T any](target *T, source []byte) error { return nil }

type FieldSpec interface{ field() }

type StringSpec struct{}

func (StringSpec) field()                           {}
func (StringSpec) Trim(...string) StringSpec        { return StringSpec{} }
func (StringSpec) NotEmpty() StringSpec             { return StringSpec{} }
func (StringSpec) Transform(string) (string, error) { return "", nil }

type NumberSpec[N ~int] struct{}

func (NumberSpec[N]) field()                 {}
func (NumberSpec[N]) Min(N) NumberSpec[N]    { return NumberSpec[N]{} }
func (NumberSpec[N]) Transform(N) (N, error) { var zero N; return zero, nil }

func String(...string) StringSpec   { return StringSpec{} }
func Int(...string) NumberSpec[int] { return NumberSpec[int]{} }
func New[T any](...FieldSpec) int   { return 0 }
