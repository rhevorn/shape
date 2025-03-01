package analyzer

import (
	"go/ast"
	"go/constant"
	"go/types"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rhevorn/shape/internal/taglang"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "shapevet",
	Doc:      "checks Shape struct tags and explicit schema fields",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Preorder([]ast.Node{(*ast.StructType)(nil), (*ast.CallExpr)(nil)}, func(node ast.Node) {
		if call, ok := node.(*ast.CallExpr); ok {
			checkConstructionCall(pass, call)
			return
		}
		st := node.(*ast.StructType)
		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}
			literal, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				continue
			}
			text, ok := reflect.StructTag(literal).Lookup("shape")
			if !ok {
				continue
			}
			items, err := taglang.Parse(text)
			if err != nil {
				pass.Reportf(field.Tag.Pos(), "invalid shape tag: %v", err)
				continue
			}
			typ := pass.TypesInfo.TypeOf(field.Type)
			if typ == nil {
				continue
			}
			if err = check(typ, items); err != nil {
				pass.Reportf(field.Tag.Pos(), "invalid shape tag: %v", err)
			}
		}
	})
	return nil, nil
}

func checkConstructionCall(pass *analysis.Pass, call *ast.CallExpr) {
	base := call.Fun
	var typeArg ast.Expr
	switch indexed := call.Fun.(type) {
	case *ast.IndexExpr:
		base, typeArg = indexed.X, indexed.Index
	case *ast.IndexListExpr:
		base = indexed.X
		if len(indexed.Indices) != 0 {
			typeArg = indexed.Indices[0]
		}
	}
	selector, ok := base.(*ast.SelectorExpr)
	if !ok {
		return
	}
	object, ok := pass.TypesInfo.Uses[selector.Sel].(*types.Func)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != "github.com/rhevorn/shape" {
		return
	}
	var target types.Type
	switch selector.Sel.Name {
	case "New":
		if typeArg != nil {
			target = pass.TypesInfo.TypeOf(typeArg)
		}
		if target != nil {
			checkExplicitSchema(pass, call, target)
		}
		return
	case "Struct":
		if typeArg != nil {
			target = pass.TypesInfo.TypeOf(typeArg)
		}
	case "BindJSON", "BindJSONReader":
		target = bindTargetType(pass, call, 0)
	case "BindJSONContext", "BindJSONReaderContext":
		target = bindTargetType(pass, call, 1)
	default:
		return
	}
	if target == nil {
		return
	}
	if err := checkStructType(target, map[types.Type]bool{}); err != nil {
		pass.Reportf(call.Pos(), "invalid Shape struct schema: %v", err)
	}
}

func checkExplicitSchema(pass *analysis.Pass, call *ast.CallExpr, target types.Type) {
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
			continue // Variables and non-standard expressions remain runtime-checked.
		}
		if len(factory.Args) == 0 {
			pass.Reportf(factory.Pos(), "invalid Shape explicit field: field name must not be empty")
			continue
		}
		nameOfFactory := calledName(pass, factory.Fun)
		if scalarFactory(nameOfFactory) && len(factory.Args) > 1 {
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
		contract := transformInputType(pass.TypesInfo.TypeOf(argument))
		if calledName(pass, factory.Fun) == "Field" && len(factory.Args) > 1 {
			contract = transformInputType(pass.TypesInfo.TypeOf(factory.Args[1]))
		}
		if contract != nil && !types.Identical(field.Type(), contract) {
			pass.Reportf(factory.Pos(), "invalid Shape explicit field: field %s has type %s, contract has type %s", name, field.Type(), contract)
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
	selector, ok := unindex(expression).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	object, _ := pass.TypesInfo.Uses[selector.Sel].(*types.Func)
	return object
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
	for i := 0; i < owner.NumFields(); i++ {
		field := owner.Field(i)
		if field.Name() == name && !field.Embedded() {
			return field
		}
	}
	return nil
}

func fieldIndex(owner *types.Struct, target *types.Var) int {
	for i := 0; i < owner.NumFields(); i++ {
		if owner.Field(i) == target {
			return i
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

func bindTargetType(pass *analysis.Pass, call *ast.CallExpr, index int) types.Type {
	if index >= len(call.Args) {
		return nil
	}
	target := types.Unalias(pass.TypesInfo.TypeOf(call.Args[index]))
	pointer, ok := target.Underlying().(*types.Pointer)
	if !ok {
		return nil
	}
	return pointer.Elem()
}

func checkStructType(t types.Type, active map[types.Type]bool) error {
	t = types.Unalias(t)
	if classify(t) == kTime {
		return errText("target must be an ordinary value struct")
	}
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	value, ok := t.Underlying().(*types.Struct)
	if !ok {
		return errText("target must be a value struct")
	}
	if active[t] {
		return errText("recursive type is unsupported")
	}
	active[t] = true
	defer delete(active, t)
	names := make(map[string]bool, value.NumFields())
	for i := 0; i < value.NumFields(); i++ {
		field := value.Field(i)
		tag := reflect.StructTag(value.Tag(i))
		shapeTag := tag.Get("shape")
		jsonName := strings.Split(tag.Get("json"), ",")[0]
		if !field.Exported() || jsonName == "-" {
			if shapeTag != "" {
				return errText(field.Name() + " has a tag but is not processed")
			}
			continue
		}
		if field.Embedded() {
			return errText("anonymous fields are unsupported")
		}
		if jsonName == "" {
			jsonName = field.Name()
		}
		if names[jsonName] {
			return errText("duplicate field name " + jsonName)
		}
		names[jsonName] = true
		items, err := taglang.Parse(shapeTag)
		if err != nil {
			return errText("field " + field.Name() + ": " + err.Error())
		}
		if err := check(field.Type(), items); err != nil {
			return errText("field " + field.Name() + ": " + err.Error())
		}
		if err := checkSupportedGraph(field.Type(), active); err != nil {
			return errText("field " + field.Name() + ": " + err.Error())
		}
	}
	return nil
}

func checkSupportedGraph(t types.Type, active map[types.Type]bool) error {
	t = types.Unalias(t)
	switch value := t.(type) {
	case *types.Pointer:
		element := types.Unalias(value.Elem())
		if _, nested := element.(*types.Pointer); nested {
			return errText("multi-level pointers are unsupported in tags")
		}
		if classify(element) == kSlice || classify(element) == kMap {
			return errText("pointers to slices and maps are unsupported in tags")
		}
		return checkSupportedGraph(element, active)
	case *types.Named:
		if classify(value) == kTime || classify(value) == kDuration || classify(value) == kString || classify(value) == kBool || classify(value) == kNumber {
			return nil
		}
		return checkStructType(value, active)
	}
	switch value := t.Underlying().(type) {
	case *types.Struct:
		return checkStructType(value, active)
	case *types.Slice:
		return checkSupportedGraph(value.Elem(), active)
	case *types.Map:
		keyKind := classify(value.Key())
		if keyKind != kString && keyKind != kNumber {
			return errText("map key must be string or integer")
		}
		return checkSupportedGraph(value.Elem(), active)
	case *types.Basic:
		if classify(value) == kUnsupported {
			return errText("unsupported field type")
		}
		return nil
	default:
		return errText("unsupported field type")
	}
}

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
	if p, ok := t.(*types.Pointer); ok {
		_ = p
		return kPointer
	}
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj != nil && obj.Pkg() != nil {
			path := obj.Pkg().Path()
			if path == "time" && obj.Name() == "Time" {
				return kTime
			}
			if path == "github.com/rhevorn/shape/types" && obj.Name() == "Duration" {
				return kDuration
			}
		}
		t = named.Underlying()
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		if info&types.IsString != 0 {
			return kString
		}
		if info&types.IsBoolean != 0 {
			return kBool
		}
		if info&(types.IsInteger|types.IsFloat) != 0 && u.Kind() != types.Uintptr {
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
func element(t types.Type) types.Type {
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}
func check(t types.Type, items []taglang.Item) error {
	k := classify(t)
	if k == kUnsupported {
		return errText("unsupported field type")
	}
	target := k
	if k == kPointer {
		target = classify(element(t))
		if target == kUnsupported || target == kPointer || target == kSlice || target == kMap {
			return errText("unsupported pointer field type")
		}
	}
	fallback, label := false, false
	for _, item := range items {
		arg := item.HasValue
		switch item.Name {
		case "label":
			if !arg || label {
				return errText("label requires one value and may appear once")
			}
			label = true
		case "ifzero":
			if !arg || fallback {
				return errText("invalid or duplicate fallback")
			}
			if k == kPointer || k == kStruct || k == kSlice || k == kMap {
				return errText("ifzero literal is unsupported for this type")
			}
			fallback = true
		case "ifnull":
			if !arg || fallback || k != kPointer || target == kStruct {
				return errText("ifnull requires a pointer to scalar, time, or duration")
			}
			fallback = true
		case "notnull":
			if arg || !(k == kPointer || k == kSlice || k == kMap) {
				return errText("notnull requires pointer, slice, or map")
			}
		case "notempty":
			if arg || !(k == kString || k == kPointer || k == kSlice || k == kMap) {
				return errText("notempty requires string, pointer, slice, or map and takes no value")
			}
		case "trim", "ltrim", "rtrim":
			if target != kString {
				return errText(item.Name + " requires string")
			}
		case "tolower", "toupper", "email", "url", "uuid", "ip":
			if arg || target != kString {
				return errText(item.Name + " requires string and takes no value")
			}
		case "minlength", "maxlength", "pattern", "startswith", "endswith", "contains":
			if !arg || target != kString {
				return errText(item.Name + " requires a string value")
			}
		case "len":
			if !arg || !(target == kString || k == kSlice || k == kMap) {
				return errText("len requires string, slice, or map")
			}
		case "min", "max":
			if !arg || !(target == kNumber || target == kDuration || k == kSlice || k == kMap) {
				return errText(item.Name + " requires number, duration, slice, or map")
			}
		case "between", "gt", "gte", "lt", "lte":
			if !arg || !(target == kNumber || target == kDuration) {
				return errText(item.Name + " requires number or duration")
			}
		case "oneof":
			if !arg || !(target == kString || target == kNumber || target == kDuration) {
				return errText("oneof requires string, number, or duration")
			}
		case "positive", "negative", "nonnegative":
			if arg || !(target == kNumber || target == kDuration) {
				return errText(item.Name + " requires number or duration and takes no value")
			}
		case "unique":
			if arg || k != kSlice {
				return errText("unique requires slice and takes no value")
			}
		default:
			return errText("unknown option " + item.Name)
		}
		if item.HasValue {
			valueType := t
			if k == kPointer {
				valueType = element(t)
			}
			if err := checkValue(valueType, target, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkValue(typ types.Type, target kind, item taglang.Item) error {
	switch item.Name {
	case "label", "trim", "ltrim", "rtrim", "startswith", "endswith", "contains":
		return nil
	case "pattern":
		if item.Value == "" {
			return errText("pattern must not be empty")
		}
		if _, err := regexp.Compile(item.Value); err != nil {
			return errText("invalid pattern: " + err.Error())
		}
		return nil
	case "minlength", "maxlength", "len":
		if _, err := strconv.ParseUint(item.Value, 10, 63); err != nil {
			return errText("invalid non-negative length")
		}
		return nil
	}
	if item.Name == "ifzero" || item.Name == "ifnull" {
		switch target {
		case kString:
			return nil
		case kBool:
			if item.Value != "true" && item.Value != "false" {
				return errText("boolean fallback must be true or false")
			}
			return nil
		case kDuration:
			if item.Value == "0" {
				return nil
			}
			if _, err := time.ParseDuration(item.Value); err != nil {
				return errText("invalid duration " + strconv.Quote(item.Value))
			}
			return nil
		case kTime:
			if _, err := time.Parse(time.RFC3339Nano, item.Value); err != nil {
				return errText("invalid RFC3339 time")
			}
			return nil
		case kNumber:
			return checkNumber(typ, item.Value)
		}
	}
	if (target == kSlice || target == kMap) && (item.Name == "min" || item.Name == "max") {
		if _, err := strconv.ParseUint(item.Value, 10, 63); err != nil {
			return errText("invalid non-negative collection length")
		}
		return nil
	}
	parts := []string{item.Value}
	if item.Name == "between" || item.Name == "oneof" {
		parts = strings.Split(item.Value, "|")
		if item.Name == "between" && len(parts) != 2 || item.Name == "oneof" && len(parts) == 0 {
			return errText("invalid list arity")
		}
	}
	if item.Name == "between" && len(parts) == 2 && greaterValue(target, parts[0], parts[1]) {
		return errText("between minimum exceeds maximum")
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return errText("empty list value")
		}
		if target == kDuration {
			if part != "0" {
				if _, err := time.ParseDuration(part); err != nil {
					return errText("invalid duration " + strconv.Quote(part))
				}
			}
			continue
		}
		if target == kString {
			continue
		}
		if target == kNumber {
			if err := checkNumber(typ, part); err != nil {
				return err
			}
		}
	}
	return nil
}

func greaterValue(target kind, left, right string) bool {
	left, right = strings.TrimSpace(left), strings.TrimSpace(right)
	if target == kDuration {
		a, aErr := time.ParseDuration(left)
		b, bErr := time.ParseDuration(right)
		return aErr == nil && bErr == nil && a > b
	}
	a, aOK := new(big.Float).SetString(left)
	b, bOK := new(big.Float).SetString(right)
	return aOK && bOK && a.Cmp(b) > 0
}

func checkNumber(typ types.Type, text string) error {
	typ = types.Unalias(typ)
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok {
		return errText("not a number")
	}
	bits := 64
	switch basic.Kind() {
	case types.Int8, types.Uint8:
		bits = 8
	case types.Int16, types.Uint16:
		bits = 16
	case types.Int32, types.Uint32, types.Float32:
		bits = 32
	}
	var err error
	if basic.Info()&types.IsUnsigned != 0 {
		_, err = strconv.ParseUint(text, 10, bits)
	} else if basic.Info()&types.IsInteger != 0 {
		_, err = strconv.ParseInt(text, 10, bits)
	} else {
		if strings.ContainsAny(text, "xXpP_") {
			err = errText("not a decimal float")
		} else {
			var value float64
			value, err = strconv.ParseFloat(text, bits)
			if err == nil && (value != value || value > 1.7976931348623157e308 || value < -1.7976931348623157e308) {
				err = errText("non-finite number")
			}
		}
	}
	if err != nil {
		return errText("invalid or overflowing number " + strconv.Quote(text))
	}
	return nil
}

type errText string

func (e errText) Error() string { return string(e) }
