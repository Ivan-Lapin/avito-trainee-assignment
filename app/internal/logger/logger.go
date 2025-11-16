package logger

import "go.uber.org/zap"

// New returns structured production logger with sane defaults.
func New() *zap.Logger {
	l, _ := zap.NewProduction()
	return l
}
