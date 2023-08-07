package multichecker

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var ErrCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check for os.Exit from main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}
	var funcDecl ast.FuncDecl
	exitFunc := ast.NewIdent("Exit")
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if currentFunc, ok := node.(*ast.FuncDecl); ok {
				funcDecl = *currentFunc
			}
			if funcDecl.Name.Name != "main" {
				return false
			}
			if exprStmt, ok := node.(*ast.CallExpr); ok {
				if exprStmt.Fun == exitFunc {

				}
			}
			return true
		})
	}

	return nil, nil
}
