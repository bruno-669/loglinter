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

// checkCall проверяет один вызов функции.
func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	// Получаем объект функции (что именно вызывается)
	msgIndex, ok := isLogFunction(pass, call)
	if !ok {
		return // не лог-функция
	}

	// Проверяем, что аргумент с сообщением существует
	if msgIndex >= len(call.Args) {
		return
	}

	// Проверяем, что аргумент — строковой литерал
	arg := call.Args[msgIndex]
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		// Не литерал — пропускаем (можно сообщить, но усложнять не будем)
		return
	}

	// Удаляем кавычки
	msg := strings.Trim(lit.Value, `"`)

	// Применяем все правила
	checkFirstLower(pass, arg.Pos(), msg, lit)
	checkOnlyEnglish(pass, arg.Pos(), msg)
	checkNoSpecialChars(pass, arg.Pos(), msg)
	checkNoSensitive(pass, arg.Pos(), msg)
}

// isLogFunction определяет, является ли вызов функцией логирования,
// и возвращает индекс аргумента, содержащего сообщение (обычно 0).
func isLogFunction(pass *analysis.Pass, call *ast.CallExpr) (int, bool) {
	// Получаем объект функции через TypesInfo
	var fn *types.Func
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// Прямой вызов функции, например slog.Info (но в AST это будет SelectorExpr, если пакет указан)
		// Для простоты будем обрабатывать только SelectorExpr, но можно и Ident, если функция импортирована как .
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
		// Вызов вида pkg.Func или obj.Method
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

	// Проверяем принадлежность к нужным пакетам
	pkg := fn.Pkg()
	if pkg == nil {
		return 0, false
	}
	pkgPath := pkg.Path()

	// Имя функции (метода)
	funcName := fn.Name()

	// Список допустимых имён функций логирования
	logFuncs := map[string]bool{
		"Info":  true,
		"Error": true,
		"Debug": true,
		"Warn":  true,
	}
	if !logFuncs[funcName] {
		return 0, false
	}

	// Проверяем пакет
	switch pkgPath {
	case "log/slog":
		// Для slog сообщение всегда первый аргумент
		return 0, true
	case "go.uber.org/zap":
		// Для zap есть несколько вариантов:
		// - методы *zap.Logger: Info(msg string, fields ...Field)
		// - методы *zap.SugaredLogger: Info(msg string, args ...interface{})
		// В обоих случаях сообщение — первый аргумент.
		// Также есть пакетные функции (zap.Info), но они редкость.
		// Упростим: считаем, что если функция из zap и имя подходит, то сообщение первый аргумент.
		return 0, true
	default:
		return 0, false
	}
}

// checkFirstLower проверяет, что сообщение начинается со строчной буквы.
func checkFirstLower(pass *analysis.Pass, pos token.Pos, msg string, originalLit *ast.BasicLit) {
	if msg == "" {
		return
	}
	first := rune(msg[0])
	if unicode.IsLower(first) {
		return
	}

	// Формируем сообщение об ошибке
	msgErr := "log message should start with a lowercase letter"

	// Генерируем исправление: заменяем первый символ на строчный
	newFirst := unicode.ToLower(first)
	// Позиция первого символа внутри литерала:
	// lit.Pos() - это позиция открывающей кавычки. Сама строка начинается с lit.Pos()+1.
	startPos := originalLit.Pos() + 1 // позиция первого символа строки
	// Заменяем только один символ
	edit := analysis.TextEdit{
		Pos:     startPos,
		End:     startPos + 1, // заменяем один символ
		NewText: []byte(string(newFirst)),
	}

	// Создаём диагностику с фиксом
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

// checkOnlyEnglish проверяет, что сообщение содержит только латинские буквы, цифры и пробелы.
func checkOnlyEnglish(pass *analysis.Pass, pos token.Pos, msg string) {
	for _, r := range msg {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ') {
			pass.Reportf(pos, "log message should contain only English letters, digits and spaces")
			return
		}
	}
}

// checkNoSpecialChars проверяет отсутствие спецсимволов и эмодзи (фактически дублирует предыдущее,
// но оставим для явного выполнения требования).
func checkNoSpecialChars(pass *analysis.Pass, pos token.Pos, msg string) {
	for _, r := range msg {
		// Разрешены буквы, цифры, пробел. Всё остальное — спецсимволы.
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
