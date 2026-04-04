package testdata

import "log/slog"

func LogSanitized(userID string) {
	slog.Info("user login", "user", userID, "token", "[REDACTED]")
}
