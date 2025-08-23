package analyzer

import (
	"go/types"
	"reflect"

	"github.com/rhevorn/shape/internal/fieldmeta"
)

func checkParameterStruct(t types.Type, source fieldmeta.Source, active map[types.Type]bool, depth int) error {
	t = types.Unalias(t)
	owner, ok := underlyingStruct(t)
	if !ok || classify(t) == kTime {
		return errText("target must be an ordinary value struct")
	}
	if active[t] || depth >= 64 {
		return errText("recursive or excessively deep struct")
	}
	active[t] = true
	defer delete(active, t)
	names := make(map[string]bool)
	for i := 0; i < owner.NumFields(); i++ {
		field := owner.Field(i)
		if !field.Exported() {
			continue
		}
		name := fieldmeta.Resolve(field.Name(), reflect.StructTag(owner.Tag(i))).For(source)
		if name == "" {
			continue
		}
		if field.Embedded() {
			return errText("anonymous fields are unsupported")
		}
		if !fieldmeta.ValidInputName(name) {
			return errText("invalid field name " + name)
		}
		if names[name] {
			return errText("duplicate field name " + name)
		}
		names[name] = true
		typ := field.Type()
		base := typ
		if classify(base) == kPointer {
			base = element(base)
		}
		var err error
		if classify(base) == kStruct && !hasTextDecoder(base) {
			err = checkParameterStruct(base, source, active, depth+1)
		} else {
			if classify(typ) == kSlice && !hasTextDecoder(typ) {
				typ = types.Unalias(typ).Underlying().(*types.Slice).Elem()
			}
			err = checkParameterScalar(typ)
		}
		if err != nil {
			return errText("field " + name + ": " + err.Error())
		}
	}
	return nil
}

func checkParameterScalar(t types.Type) error {
	if classify(t) == kPointer {
		inner := element(t)
		if k := classify(inner); k == kPointer || k == kSlice || k == kMap {
			return errText("unsupported pointer parameter type")
		}
		return checkParameterScalar(inner)
	}
	if hasTextDecoder(t) {
		return nil
	}
	switch classify(t) {
	case kString, kBool, kNumber, kTime, kDuration:
		return nil
	default:
		return errText("unsupported parameter type")
	}
}

func hasTextDecoder(t types.Type) bool {
	method := types.NewMethodSet(types.NewPointer(t)).Lookup(nil, "UnmarshalText")
	if method == nil {
		return false
	}
	signature, ok := method.Obj().Type().(*types.Signature)
	if !ok || signature.Variadic() || signature.Params().Len() != 1 || signature.Results().Len() != 1 {
		return false
	}
	return types.Identical(signature.Params().At(0).Type(), types.NewSlice(types.Typ[types.Byte])) && types.Identical(signature.Results().At(0).Type(), types.Universe.Lookup("error").Type())
}
