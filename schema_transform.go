package shape

import (
	"reflect"

	"github.com/rhevorn/shape/transform"
)

// schemaTransformer unwraps exact built-in value Specs. A promoted method on
// an external wrapper does not establish that its public behavior is unchanged.
func schemaTransformer[T any](schema Schema[T]) transform.Transformer[T] {
	actual := reflect.TypeOf(schema)
	base := actual
	if base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if base.PkgPath() != reflect.TypeFor[StringSpec]().PkgPath() {
		return schema
	}
	if provider, ok := schema.(interface {
		schemaTransformer() (reflect.Type, transform.Transformer[T])
	}); ok {
		typ, transformer := provider.schemaTransformer()
		if actual == typ || actual == reflect.PointerTo(typ) {
			return transformer
		}
	}
	return schema
}

func (s ValueSpec[T]) schemaTransformer() (reflect.Type, transform.Transformer[T]) {
	return reflect.TypeFor[ValueSpec[T]](), s.transformer
}
func (s StringSpec) schemaTransformer() (reflect.Type, transform.Transformer[string]) {
	return reflect.TypeFor[StringSpec](), s.transformer
}
func (s NumberSpec[N]) schemaTransformer() (reflect.Type, transform.Transformer[N]) {
	return reflect.TypeFor[NumberSpec[N]](), s.transformer
}
func (s PointerSpec[T]) schemaTransformer() (reflect.Type, transform.Transformer[*T]) {
	return reflect.TypeFor[PointerSpec[T]](), s.transformer
}
func (s SliceSpec[T]) schemaTransformer() (reflect.Type, transform.Transformer[[]T]) {
	return reflect.TypeFor[SliceSpec[T]](), s.transformer
}
func (s MapSpec[K, V]) schemaTransformer() (reflect.Type, transform.Transformer[map[K]V]) {
	return reflect.TypeFor[MapSpec[K, V]](), s.transformer
}
