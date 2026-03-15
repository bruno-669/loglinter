# Линтер для проверки лог-записей loglinter

loglinter – статический анализатор кода на Go, проверяющий сообщения в вызовах функций логирования на соответствие заданным правилам стиля и безопасности. Анализатор может запускаться как отдельная утилита, так и интегрироваться в golangci-lint в качестве плагина.

## Функциональность

Линтер анализирует вызовы функций Info, Error, Debug, Warn из пакетов:

- `log/slog`
- `go.uber.org/zap` (методы Logger и SugaredLogger)

Для каждого строкового аргумента, передаваемого в лог-функцию в качестве сообщения, применяются следующие проверки.

* Начало сообщения со строчной буквы
	Сообщение должно начинаться со строчной буквы (символ в нижнем регистре). Заглавная первая буква считается нарушением.

```go
// некорректный ввод
slog.Info("Starting server")   // первая буква 'S' заглавная
zap.L().Error("Failed to connect")
// корректный ввод
slog.Info("starting server")
zap.L().Error("failed to connect")
```
	
* Допустимые символы: только латиница, цифры и пробел
	Сообщение может содержать только английские буквы (A–Z, a–z), цифры (0–9) и пробелы. Любые другие символы (кириллица, знаки пунктуации, математические символы и т.д.) считаются недопустимыми.

```go
// некорректный ввод
slog.Info("запуск сервера")      // кириллица
slog.Debug("значение: 42")       // кириллица и двоеточие
// корректный ввод
slog.Info("server started")
slog.Debug("value 42")
```

* Запрет спецсимволов и эмодзи
	Правило фактически дублирует предыдущее, но выделено для явного соответствия требованию. Любой символ, не являющийся латинской буквой, цифрой или пробелом, классифицируется как спецсимвол или эмодзи и приводит к диагностике.

```go
// некорректный ввод
slog.Warn("connection failed!!")   // восклицательные знаки
slog.Info("✅ success")              // эмодзи
// корректный ввод
slog.Warn("connection failed")
slog.Info("success")
```
*  Отсутствие чувствительных данных

	Линтер проверяет наличие в сообщении ключевых слов, указывающих на потенциально конфиденциальную информацию. Поиск выполняется без учёта регистра. Список ключевых слов:

	- `password`, `passwd`, `pwd`
	- `token`
	- `api_key`, `apikey`
	- `secret`
	- `key`
	- `auth`
	- `credential`

```go
// некорректный ввод
slog.Info("user password: 12345")
slog.Debug("api_key=abc123")
zap.L().Info("token: eyJhbGci...")
// корректный ввод
slog.Info("user authenticated")
slog.Debug("api request completed")
zap.L().Info("token validated")
```
## Автоисправление (Suggested Fixes)

Для правила «Начало сообщения со строчной буквы» реализовано автоматическое исправление. Линтер предлагает заменить первый символ сообщения на соответствующий символ в нижнем регистре.

При использовании отдельной утилиты исправления можно применить с помощью флага `-fix`:

```bash
loglinter -fix ./...
```

В составе `golangci-lint` исправления применяются при запуске с флагом `--fix`:

```bash
golangci-lint run --fix
```

Исправление вносит изменения непосредственно в исходный код: заменяет только первый символ строкового литерала, сохраняя остальную часть сообщения.

## Конфигурация

### Настройка списка чувствительных слов

Через флаг `-sensitive-words` можно передать собственный перечень ключевых слов (через запятую). Пробелы вокруг запятых игнорируются.

Пример запуска отдельной утилиты:
```bash
loglinter -sensitive-words=pass,token,secret,credential ./...
```

### Интеграция с golangci-lint

В файле `.golangci.yml` можно указать путь к плагину и передать ему флаги:

```yaml
linters-settings:
  custom:
    loglinter:
      path: ./plugin/loglinter.so   # путь к собранному плагину
      description: Линтер для проверки логов
      original-url: github.com/bruno-669/loglinter
      flags:
        - -sensitive-words=password,token,api_key,secret,auth,credential
linters:
  enable:
    - loglinter
```

Пример актуального файла конфигурации можно найти в корне репозитория (`.golangci.yml`).

## Установка

### Как отдельная утилита

Требуется Go версии 1.22 или выше. Выполните:

```bash
go install github.com/bruno-669/loglinter/cmd/loglinter@latest
```

После установки исполняемый файл `loglinter` будет доступен в `$GOPATH/bin` (или `$HOME/go/bin`, если `GOPATH` не задан).

### Как плагин для golangci-lint

1. Клонируйте репозиторий:
        
    ```bash
    git clone https://github.com/bruno-669/loglinter.git
    cd loglinter
    ```
    
1. Соберите плагин:
    ```bash
    go build -buildmode=plugin -o plugin/loglinter.so ./cmd/loglinter
    ```
    
    Будет создан файл `plugin/loglinter.so`.
    
2. Настройте `golangci-lint` как описано в разделе «Конфигурация».
    

## Использование

### Запуск отдельной утилиты

Команда `loglinter` принимает те же аргументы, что и стандартные инструменты `go`:

```bash
loglinter [флаги] [пакеты]

Примеры:

loglinter .                           //Проверить пакет в текущем каталоге
loglinter ./...                       //Проверить все пакеты модуля
loglinter main.go                     //Проверить конкретные файлы
loglinter -fix -sensitive-words=pass,token,key ./... //Запустить с автоисправлением и собственным списком чувствительных слов
```

Линтер выводит диагностические сообщения в формате:

```
файл:строка:столбец: сообщение об ошибке
```

### Запуск в составе golangci-lint

После настройки плагина линтер будет автоматически вызываться при запуске `golangci-lint run`. Вывод соответствует формату `golangci-lint`. Для применения исправлений используйте `golangci-lint run --fix`.

## Тестирование

В репозитории реализованы модульные тесты, использующие пакет `golang.org/x/tools/go/analysis/analysistest`. Тесты находятся в файле `analyzer_test.go`, а тестовые данные — в каталоге `testdata/`.

- `testdata/src/` содержит примеры кода с ожидаемыми диагностиками.
    
- `testdata/fix/` содержит примеры для проверки автоисправления (исходный файл `fix.go` и эталонный `fix.go.golden`).
    

Запуск тестов:

```bash
go test -v ./...
```

Все тесты должны проходить успешно.
## Требования к окружению

- Go версии 1.22 или новее.
    
- Для сборки плагина требуется поддержка `-buildmode=plugin` (доступна на большинстве платформ).
    
- При использовании с `golangci-lint` необходима версия, поддерживающая пользовательские плагины (golangci-lint v1.45.0 и выше).
    

## Пример работы

Исходный файл `example.go`:
```go
package main
import (
    "log/slog"
    "go.uber.org/zap"
)
func main() {
    slog.Info("Starting server on port 8080")
    slog.Error("Failed to connect to database")
    slog.Info("запуск сервера")
    slog.Warn("warning!")
    slog.Debug("user password: secret")
}
```

Запуск линтера:
```bash
$ loglinter example.go
example.go:8:2: log message should start with a lowercase letter
example.go:9:2: log message should start with a lowercase letter
example.go:10:2: log message should contain only English letters, digits and spaces
example.go:11:2: log message should not contain special characters or emojis
example.go:12:2: log message should not contain sensitive data (found "password")
```

Запуск с автоисправлением:
```bash
$ loglinter -fix example.go
```

После исправления файл будет изменён: первая буква в сообщениях станет строчной.

---

_Документация соответствует версии анализатора, реализованной в репозитории. Все описанные правила, возможность настройки чувствительных слов и автоисправление поддерживаются в текущей реализации._