package testdata

import "log/slog"

func LogStructured() {
	slog.Info("request complete", "status", 200, "component", "api")
}
