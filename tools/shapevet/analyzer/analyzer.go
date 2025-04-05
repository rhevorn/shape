// Package analyzer implements the shapevet static analyzer.
package analyzer

import (
	"go/ast"
	"go/types"
	"reflect"
	"strconv"
	"strings"

	"github.com/rhevorn/shape/internal/taglang"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer checks Shape struct tags and explicit schema field declarations.
var Analyzer = &analysis.Analyzer{
	Name:     "shapevet",
	Doc:      "checks Shape struct tags and explicit schema fields",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var excludeTestFiles bool

func init() {
	Analyzer.Flags.BoolVar(&excludeTestFiles, "exclude-tests", false, "skip files ending in _test.go")
}

func run(pass *analysis.Pass) (any, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Preorder([]ast.Node{(*ast.StructType)(nil), (*ast.CallExpr)(nil)}, func(node ast.Node) {
		if excludeTestFiles && strings.HasSuffix(pass.Fset.Position(node.Pos()).Filename, "_test.go") {
			return
		}
		if call, ok := node.(*ast.CallExpr); ok {
			checkConstructionCall(pass, call)
			return
		}
		structure := node.(*ast.StructType)
		for _, field := range structure.Fields.List {
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
			if typ == nil || mentionsTypeParam(typ) {
				continue
			}
			if err = checkTag(typ, items); err != nil {
				pass.Reportf(field.Tag.Pos(), "invalid shape tag: %v", err)
			}
		}
	})
	return nil, nil
}

func checkConstructionCall(pass *analysis.Pass, call *ast.CallExpr) {
	base := call.Fun
	var typeArgument ast.Expr
	switch indexed := call.Fun.(type) {
	case *ast.IndexExpr:
		base, typeArgument = indexed.X, indexed.Index
	case *ast.IndexListExpr:
		base = indexed.X
		if len(indexed.Indices) != 0 {
			typeArgument = indexed.Indices[0]
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
		if typeArgument != nil {
			target = pass.TypesInfo.TypeOf(typeArgument)
		}
		if target != nil {
			checkExplicitSchema(pass, call, target)
		}
		return
	case "Struct":
		if typeArgument != nil {
			target = pass.TypesInfo.TypeOf(typeArgument)
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
	if _, generic := types.Unalias(target).(*types.TypeParam); generic {
		return
	}
	if err := checkStructType(target, map[types.Type]bool{}); err != nil {
		pass.Reportf(call.Pos(), "invalid Shape struct schema: %v", err)
	}
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
