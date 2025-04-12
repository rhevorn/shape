package analyzer

import (
	"go/ast"
	"go/constant"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func checkExplicitSchema(pass *analysis.Pass, call *ast.CallExpr, target types.Type) {
	if mentionsTypeParam(target) {
		return
	}
	if classify(target) == kTime {
		pass.Reportf(call.Pos(), "invalid Shape explicit schema: target must be an ordinary value struct")
		return
	}
	owner, ok := underlyingStruct(target)
	if !ok {
		pass.Reportf(call.Pos(), "invalid Shape explicit schema: target must be a value struct")
		return
	}
	seen := make(map[string]bool, len(call.Args))
	for _, argument := range call.Args {
		factory := explicitFactoryCall(pass, argument)
		if factory == nil {
			continue
		}
		if len(factory.Args) == 0 {
			pass.Reportf(factory.Pos(), "invalid Shape explicit field: field name must not be empty")
			continue
		}
		factoryName := calledName(pass, factory.Fun)
		if scalarFactory(factoryName) && len(factory.Args) > 1 {
			pass.Reportf(factory.Args[1].Pos(), "invalid Shape explicit field: field factory accepts at most one name")
			continue
		}
		name, ok := constantString(pass, factory.Args[0])
		if !ok {
			continue
		}
		if name == "" {
			pass.Reportf(factory.Args[0].Pos(), "invalid Shape explicit field: field name must not be empty")
			continue
		}
		if seen[name] {
			pass.Reportf(factory.Args[0].Pos(), "invalid Shape explicit field: duplicate field %s", name)
			continue
		}
		seen[name] = true
		field := directField(owner, name)
		if field == nil {
			pass.Reportf(factory.Args[0].Pos(), "invalid Shape explicit field: target has no direct field %s", name)
			continue
		}
		if !field.Exported() {
			pass.Reportf(factory.Args[0].Pos(), "invalid Shape explicit field: field %s is not exported", name)
			continue
		}
		jsonName := reflect.StructTag(owner.Tag(fieldIndex(owner, field))).Get("json")
		if strings.Split(jsonName, ",")[0] == "-" {
			pass.Reportf(factory.Args[0].Pos(), "invalid Shape explicit field: field %s is excluded from JSON", name)
			continue
		}
		schemaType := transformInputType(pass.TypesInfo.TypeOf(argument))
		if factoryName == "Field" && len(factory.Args) > 1 {
			schemaType = transformInputType(pass.TypesInfo.TypeOf(factory.Args[1]))
		}
		if schemaType != nil && !types.Identical(field.Type(), schemaType) {
			pass.Reportf(factory.Pos(), "invalid Shape explicit field: field %s has type %s, schema has type %s", name, field.Type(), schemaType)
		}
	}
}

func scalarFactory(name string) bool {
	switch name {
	case "Value", "String", "Number", "Int", "Int64", "Float64", "Duration", "Bool", "Time":
		return true
	default:
		return false
	}
}

func underlyingStruct(target types.Type) (*types.Struct, bool) {
	target = types.Unalias(target)
	if named, ok := target.(*types.Named); ok {
		target = named.Underlying()
	}
	value, ok := target.Underlying().(*types.Struct)
	return value, ok
}

func explicitFactoryCall(pass *analysis.Pass, expression ast.Expr) *ast.CallExpr {
	current := expression
	for {
		call, ok := current.(*ast.CallExpr)
		if !ok {
			return nil
		}
		name := calledName(pass, call.Fun)
		switch name {
		case "Value", "String", "Number", "Int", "Int64", "Float64", "Duration", "Bool", "Time", "Pointer", "Slice", "Map", "Field":
			object := calledObject(pass, call.Fun)
			if object == nil {
				break
			}
			signature, _ := object.Type().(*types.Signature)
			if object.Pkg() != nil && object.Pkg().Path() == "github.com/rhevorn/shape" && signature != nil && signature.Recv() == nil {
				return call
			}
		}
		base := unindex(call.Fun)
		selector, ok := base.(*ast.SelectorExpr)
		if !ok {
			return nil
		}
		current = selector.X
	}
}

func calledName(pass *analysis.Pass, expression ast.Expr) string {
	object := calledObject(pass, expression)
	if object == nil {
		return ""
	}
	return object.Name()
}

func calledObject(pass *analysis.Pass, expression ast.Expr) *types.Func {
	switch expression := unindex(expression).(type) {
	case *ast.SelectorExpr:
		object, _ := pass.TypesInfo.Uses[expression.Sel].(*types.Func)
		return object
	case *ast.Ident:
		object, _ := pass.TypesInfo.Uses[expression].(*types.Func)
		return object
	default:
		return nil
	}
}

func unindex(expression ast.Expr) ast.Expr {
	switch indexed := expression.(type) {
	case *ast.IndexExpr:
		return indexed.X
	case *ast.IndexListExpr:
		return indexed.X
	default:
		return expression
	}
}

func constantString(pass *analysis.Pass, expression ast.Expr) (string, bool) {
	value := pass.TypesInfo.Types[expression].Value
	if value == nil || value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(value), true
}

func directField(owner *types.Struct, name string) *types.Var {
	for index := 0; index < owner.NumFields(); index++ {
		field := owner.Field(index)
		if field.Name() == name && !field.Embedded() {
			return field
		}
	}
	return nil
}

func fieldIndex(owner *types.Struct, target *types.Var) int {
	for index := 0; index < owner.NumFields(); index++ {
		if owner.Field(index) == target {
			return index
		}
	}
	return -1
}

func transformInputType(t types.Type) types.Type {
	if t == nil {
		return nil
	}
	selection := types.NewMethodSet(t).Lookup(nil, "Transform")
	if selection == nil {
		return nil
	}
	signature, ok := selection.Obj().Type().(*types.Signature)
	if !ok || signature.Params().Len() != 1 {
		return nil
	}
	return signature.Params().At(0).Type()
}
