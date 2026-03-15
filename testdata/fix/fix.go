package fix

import "log/slog"

func main() {
	slog.Info("Server started") // want "log message should start with a lowercase letter"
}
