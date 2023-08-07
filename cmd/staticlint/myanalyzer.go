package multichecker

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check for os.Exit from main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		if file.Name.Name != "main" {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if exprStmt, ok := node.(*ast.CallExpr); ok {
				if fun, ok := exprStmt.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := fun.X.(*ast.Ident); ok {
						if ident.Name == "os" && fun.Sel.Name == "Exit" {
							pass.Reportf(exprStmt.Pos(), "call os.Exit in main function")
						}
					}

				}
			}
			return true
		})

	}

	return nil, nil
}
