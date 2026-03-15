package testdata

import (
	"log/slog"

	"go.uber.org/zap"
)

func main() {
	slog.Info("Starting server on port 8080")
	slog.Error("Failed to connect to database")
	slog.Info("запуск сервера")         // want "log message should contain only English letters, digits and spaces"
	slog.Warn("warning!")               // want "log message should not contain special characters or emojis"
	slog.Debug("user password: secret") // want "log message should not contain sensitive data"

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("starting server")
	logger.Error("failed to connect")
	logger.Debug("ошибка")             // want "log message should contain only English letters, digits and spaces"
	logger.Warn("connection failed!!") // want "log message should not contain special characters or emojis"
	logger.Info("token: abc123")       // want "log message should not contain sensitive data"
}
