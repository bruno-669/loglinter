# loglinter: A Linter for Log Messages

loglinter is a static code analyzer for Go that checks log function calls for compliance with style and security rules. The analyzer can be run as a standalone tool or integrated into `golangci-lint` as a plugin.

## Features

The linter analyzes calls to `Info`, `Error`, `Debug`, `Warn` functions from the following packages:

- `log/slog`
- `go.uber.org/zap` (both `Logger` and `SugaredLogger` methods)

For every string argument passed as a message to a logging function, the following checks are applied.

- **Message must start with a lowercase letter**  
    The message must begin with a lowercase letter. An uppercase first letter is considered a violation.

```go
// invalid
slog.Info("Starting server")   // first letter 'S' is uppercase
zap.L().Error("Failed to connect")
// valid
slog.Info("starting server")
zap.L().Error("failed to connect")
```
	
* **Allowed characters: only Latin letters, digits, and spaces**  
	The message may contain only English letters (A–Z, a–z), digits (0–9), and spaces. Any other characters (Cyrillic, punctuation marks, mathematical symbols, etc.) are considered invalid.

```go
// invalid
slog.Info("запуск сервера")      // Cyrillic
slog.Debug("значение: 42")       // Cyrillic and colon
// valid
slog.Info("server started")
slog.Debug("value 42")
```

* ***No special characters or emojis**  
	This rule essentially duplicates the previous one but is explicitly mentioned to meet requirements. Any character that is not a Latin letter, digit, or space is classified as a special character or emoji and triggers a diagnostic.

```go
// invalid
slog.Warn("connection failed!!")   // exclamation marks
slog.Info("✅ success")              // emoji
// valid
slog.Warn("connection failed")
slog.Info("success")
```
* ***No sensitive data**  
	The linter checks for keywords in the message that may indicate potentially sensitive information. The search is case-insensitive. The list of keywords includes:
	- `password`, `passwd`, `pwd`
	- `token`
	- `api_key`, `apikey`
	- `secret`
	- `key`
	- `auth`
	- `credential`

```go
// invalid
slog.Info("user password: 12345")
slog.Debug("api_key=abc123")
zap.L().Info("token: eyJhbGci...")
// valid
slog.Info("user authenticated")
slog.Debug("api request completed")
zap.L().Info("token validated")
```
## Autofix (Suggested Fixes)

For the rule "Message must start with a lowercase letter," an automatic fix is implemented. The linter suggests replacing the first character of the message with its lowercase equivalent.

When using the standalone tool, fixes can be applied with the `-fix` flag:

```bash
loglinter -fix ./...
```

When running within `golangci-lint`, fixes are applied with the `--fix` flag:

```bash
golangci-lint run --fix
```

The fix modifies the source code directly: it only changes the first character of the string literal while preserving the rest of the message.

## Configuration

### Customizing the sensitive words list

You can provide your own list of keywords (comma-separated) using the `-sensitive-words` flag. Spaces around commas are ignored.

Example of running the standalone tool:
```bash
loglinter -sensitive-words=pass,token,secret,credential ./...
```
### Integration with golangci-lint

In the `.golangci.yml` file, you can specify the path to the plugin and pass flags to it:

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

An up-to-date configuration file example can be found in the repository root (`.golangci.yml`).

## Installation

### As a standalone tool

Requires Go version 1.22 or higher. Run:

```bash
go install github.com/bruno-669/loglinter/cmd/loglinter@latest
```

After installation, the `loglinter` executable will be available in `$GOPATH/bin` (or `$HOME/go/bin` if `GOPATH` is not set).
### As a plugin for golangci-lint

1. Clone the repository:
    
```bash
git clone https://github.com/bruno-669/loglinter.git
cd loglinter
```
    
2. Build the plugin:
```go
build -buildmode=plugin -o plugin/loglinter.so ./cmd/loglinter
```
	This creates the file `plugin/loglinter.so`.
	
3. Configure `golangci-lint` as described in the "Configuration" section.
    
## Usage

### Running the standalone tool

The `loglinter` command accepts the same arguments as the standard `go` tools:

```bash
loglinter [flags] [packages]

Examples:

loglinter .                           # Check the package in the current directory
loglinter ./...                       # Check all packages in the module
loglinter main.go                      # Check specific files
loglinter -fix -sensitive-words=pass,token,key ./...   # Run with autofix and a custom sensitive words list
```

The linter outputs diagnostics in the format:

```
file:line:column: error message
```

### Running as part of golangci-lint

After configuring the plugin, the linter will be automatically invoked when running `golangci-lint run`. The output follows the `golangci-lint` format. To apply fixes, use `golangci-lint run --fix`.
## Testing

The repository includes unit tests that use the `golang.org/x/tools/go/analysis/analysistest` package. Tests are located in `analyzer_test.go`, and test data is in the `testdata/` directory.

- `testdata/src/` contains code examples with expected diagnostics.
    
- `testdata/fix/` contains examples for testing autofix (source file `fix.go` and expected `fix.go.golden`).
    

Run tests:

```bash
go test -v ./...
```

All tests should pass.

## Requirements

- Go version 1.22 or newer.
    
- Building the plugin requires support for `-buildmode=plugin` (available on most platforms).
    
- When using with `golangci-lint`, a version that supports custom plugins is required (golangci-lint v1.45.0 and above).
    
## Example

Source file `example.go`:
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

Running the linter:
```bash
$ loglinter example.go
example.go:8:2: log message should start with a lowercase letter
example.go:9:2: log message should start with a lowercase letter
example.go:10:2: log message should contain only English letters, digits and spaces
example.go:11:2: log message should not contain special characters or emojis
example.go:12:2: log message should not contain sensitive data (found "password")
```

Running with autofix:
```bash
$ loglinter -fix example.go
```
After the fix, the file will be modified so that the first letter in the messages becomes lowercase.

---

__This documentation corresponds to the version of the analyzer implemented in the repository. All described rules, the ability to customize the sensitive words list, and autofix are supported in the current implementation.__