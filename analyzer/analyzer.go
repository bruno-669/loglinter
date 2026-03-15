package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var sensitiveWords string

var Analyzer = &analysis.Analyzer{
	Name:     "loglinter",
	Doc:      "checks log messages for style and security",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func init() {
	Analyzer.Flags.StringVar(&sensitiveWords, "sensitive-words", "password,token,secret,key,auth,credential", "comma-separated list of sensitive words to check in log messages")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		checkCall(pass, call)
	})
	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr) {

	msgIndex, ok := isLogFunction(pass, call)
	if !ok {
		return
	}

	if msgIndex >= len(call.Args) {
		return
	}

	arg := call.Args[msgIndex]
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {

		return
	}

	msg := strings.Trim(lit.Value, `"`)

	checkFirstLower(pass, arg.Pos(), msg, lit)
	checkOnlyEnglish(pass, arg.Pos(), msg)
	checkNoSpecialChars(pass, arg.Pos(), msg)
	checkNoSensitive(pass, arg.Pos(), msg)
}

func isLogFunction(pass *analysis.Pass, call *ast.CallExpr) (int, bool) {

	var fn *types.Func
	switch fun := call.Fun.(type) {
	case *ast.Ident:

		obj := pass.TypesInfo.ObjectOf(fun)
		if obj == nil {
			return 0, false
		}
		var ok bool
		fn, ok = obj.(*types.Func)
		if !ok {
			return 0, false
		}
	case *ast.SelectorExpr:

		obj := pass.TypesInfo.ObjectOf(fun.Sel)
		if obj == nil {
			return 0, false
		}
		var ok bool
		fn, ok = obj.(*types.Func)
		if !ok {
			return 0, false
		}
	default:
		return 0, false
	}

	pkg := fn.Pkg()
	if pkg == nil {
		return 0, false
	}
	pkgPath := pkg.Path()

	funcName := fn.Name()

	logFuncs := map[string]bool{
		"Info":  true,
		"Error": true,
		"Debug": true,
		"Warn":  true,
	}
	if !logFuncs[funcName] {
		return 0, false
	}

	switch pkgPath {
	case "log/slog":

		return 0, true
	case "go.uber.org/zap":

		return 0, true
	default:
		return 0, false
	}
}

func checkFirstLower(pass *analysis.Pass, pos token.Pos, msg string, originalLit *ast.BasicLit) {
	if msg == "" {
		return
	}
	first := rune(msg[0])
	if unicode.IsLower(first) {
		return
	}

	msgErr := "log message should start with a lowercase letter"

	newFirst := unicode.ToLower(first)

	startPos := originalLit.Pos() + 1

	edit := analysis.TextEdit{
		Pos:     startPos,
		End:     startPos + 1,
		NewText: []byte(string(newFirst)),
	}

	pass.Report(analysis.Diagnostic{
		Pos:     pos,
		Message: msgErr,
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message:   "make first letter lowercase",
				TextEdits: []analysis.TextEdit{edit},
			},
		},
	})
}

func checkOnlyEnglish(pass *analysis.Pass, pos token.Pos, msg string) {
	for _, r := range msg {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ') {
			pass.Reportf(pos, "log message should contain only English letters, digits and spaces")
			return
		}
	}
}

func checkNoSpecialChars(pass *analysis.Pass, pos token.Pos, msg string) {
	for _, r := range msg {

		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ') {
			pass.Reportf(pos, "log message should not contain special characters or emojis")
			return
		}
	}
}

func checkNoSensitive(pass *analysis.Pass, pos token.Pos, msg string) {
	words := strings.Split(sensitiveWords, ",")
	lowerMsg := strings.ToLower(msg)
	for _, kw := range words {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if strings.Contains(lowerMsg, kw) {
			pass.Reportf(pos, "log message should not contain sensitive data (found %q)", kw)
			return
		}
	}
}
