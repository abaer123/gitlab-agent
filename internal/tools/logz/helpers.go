package logz

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func LevelFromString(levelStr string) (zapcore.Level, error) {
	var level zapcore.Level
	err := level.Set(levelStr)
	if err != nil {
		return level, fmt.Errorf("log level: %v", err)
	}
	switch level { // nolint: exhaustive
	case zap.DebugLevel, zap.InfoLevel, zap.WarnLevel, zap.ErrorLevel:
	default:
		return level, fmt.Errorf("unsupported log level: %s", level)
	}
	return level, nil
}

func LoggerWithLevel(level zapcore.LevelEnabler) *zap.Logger {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.TimeKey = "time"
	lockedSyncer := zapcore.Lock(os.Stderr)
	return zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg),
			lockedSyncer,
			level,
		),
		zap.ErrorOutput(lockedSyncer),
	)
}
