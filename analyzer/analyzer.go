package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer - это объект, который описывает наш линтер.
var Analyzer = &analysis.Analyzer{
	Name:     "loglinter",                                  // имя линтера
	Doc:      "checks log messages for style and security", // описание
	Run:      run,                                          // функция, которая выполняет проверку
	Requires: []*analysis.Analyzer{inspect.Analyzer},       // нам нужен инспектор AST
}

// run вызывается для каждого пакета, который анализируется.
func run(pass *analysis.Pass) (interface{}, error) {
	// Используем инспектор для эффективного обхода AST.
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Фильтр узлов: нас интересуют только вызовы функций.
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// Preorder обходит AST и вызывает функцию для каждого узла, подходящего под фильтр.
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		checkCall(pass, call)
	})

	return nil, nil
}

// checkCall анализирует один вызов функции.
func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	// Пока просто заглушка: будем выводить позицию вызова.
	pass.Reportf(call.Pos(), "found a function call")
}
