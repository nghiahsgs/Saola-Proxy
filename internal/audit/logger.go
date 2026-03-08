package audit

import (
	"log/slog"
	"os"
	"strings"
)

// sensitiveKeys are attribute keys whose values must never be logged.
var sensitiveKeys = map[string]bool{
	"value":    true,
	"original": true,
	"pii":      true,
	"secret":   true,
	"password": true,
	"token":    true,
}

// SetupLogger creates a JSON slog.Logger writing to stderr that drops any
// attribute whose key appears in sensitiveKeys.
func SetupLogger(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: l,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if sensitiveKeys[a.Key] {
				return slog.Attr{} // drop the attribute entirely
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(handler)
}
