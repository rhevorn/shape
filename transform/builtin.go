package transform

import "reflect"

func builtIn[T any](t Transformer[T]) (ownedTransformer[T], bool) {
	actual := reflect.TypeOf(t)
	if actual == nil {
		return nil, false
	}
	base := actual
	if base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if base.PkgPath() != reflect.TypeFor[StringTransformer]().PkgPath() {
		return nil, false
	}
	owned, ok := t.(ownedTransformer[T])
	if !ok {
		return nil, false
	}
	typ := owned.transformerType()
	return owned, actual == typ || actual == reflect.PointerTo(typ)
}

func identityTransformer[T any](t Transformer[T]) bool {
	if _, ok := builtIn(t); !ok {
		return false
	}
	identity, ok := t.(interface{ isIdentity() bool })
	return ok && identity.isIdentity()
}

func (ValueTransformer[T]) transformerType() reflect.Type {
	return reflect.TypeFor[ValueTransformer[T]]()
}
func (NumberTransformer[N]) transformerType() reflect.Type {
	return reflect.TypeFor[NumberTransformer[N]]()
}
func (StringTransformer) transformerType() reflect.Type { return reflect.TypeFor[StringTransformer]() }
func (PointerTransformer[T]) transformerType() reflect.Type {
	return reflect.TypeFor[PointerTransformer[T]]()
}
func (SliceTransformer[T]) transformerType() reflect.Type {
	return reflect.TypeFor[SliceTransformer[T]]()
}
func (MapTransformer[K, V]) transformerType() reflect.Type {
	return reflect.TypeFor[MapTransformer[K, V]]()
}
func (sequence[T]) transformerType() reflect.Type { return reflect.TypeFor[sequence[T]]() }

func (t ValueTransformer[T]) isIdentity() bool { return len(t.steps) == 0 }
func (t PointerTransformer[T]) isIdentity() bool {
	return t.elements != nil && t.identity && t.value.isIdentity()
}
func (t SliceTransformer[T]) isIdentity() bool {
	return t.elements != nil && t.identity && t.value.isIdentity()
}
func (t MapTransformer[K, V]) isIdentity() bool {
	return t.elements != nil && t.identity && t.value.isIdentity()
}
