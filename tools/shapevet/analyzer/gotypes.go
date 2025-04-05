package analyzer

import "go/types"

type kind uint8

const (
	kUnsupported kind = iota
	kString
	kBool
	kNumber
	kTime
	kDuration
	kPointer
	kSlice
	kMap
	kStruct
)

func classify(t types.Type) kind {
	t = types.Unalias(t)
	if _, ok := t.(*types.Pointer); ok {
		return kPointer
	}
	if named, ok := t.(*types.Named); ok {
		object := named.Obj()
		if object != nil && object.Pkg() != nil {
			path := object.Pkg().Path()
			if path == "time" && object.Name() == "Time" {
				return kTime
			}
			if path == "github.com/rhevorn/shape/types" && object.Name() == "Duration" {
				return kDuration
			}
		}
		t = named.Underlying()
	}
	switch underlying := t.Underlying().(type) {
	case *types.Basic:
		info := underlying.Info()
		if info&types.IsString != 0 {
			return kString
		}
		if info&types.IsBoolean != 0 {
			return kBool
		}
		if info&(types.IsInteger|types.IsFloat) != 0 && underlying.Kind() != types.Uintptr {
			return kNumber
		}
	case *types.Slice:
		return kSlice
	case *types.Map:
		return kMap
	case *types.Struct:
		return kStruct
	}
	return kUnsupported
}

func isMapKey(t types.Type) bool {
	basic, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return false
	}
	info := basic.Info()
	return info&types.IsString != 0 || info&types.IsInteger != 0 && basic.Kind() != types.Uintptr
}

func mentionsTypeParam(t types.Type) bool {
	switch value := types.Unalias(t).(type) {
	case *types.TypeParam:
		return true
	case *types.Pointer:
		return mentionsTypeParam(value.Elem())
	case *types.Slice:
		return mentionsTypeParam(value.Elem())
	case *types.Array:
		return mentionsTypeParam(value.Elem())
	case *types.Chan:
		return mentionsTypeParam(value.Elem())
	case *types.Map:
		return mentionsTypeParam(value.Key()) || mentionsTypeParam(value.Elem())
	case *types.Named:
		arguments := value.TypeArgs()
		for index := 0; index < arguments.Len(); index++ {
			if mentionsTypeParam(arguments.At(index)) {
				return true
			}
		}
	}
	return false
}

func element(t types.Type) types.Type {
	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		return pointer.Elem()
	}
	return t
}
