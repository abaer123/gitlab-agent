package logz

import (
	"fmt"
	"io"
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
	lockedSyncer := zapcore.Lock(NoSync(os.Stderr))
	return zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg),
			lockedSyncer,
			level,
		),
		zap.ErrorOutput(lockedSyncer),
	)
}

// NoSync can be used to wrap a io.Writer that implements zapcore.WriteSyncer but does not actually
// support the Sync() operation. An example is os.Stderr that returns
// "sync /dev/stderr: inappropriate ioctl for device" on sync attempt.
func NoSync(w io.Writer) zapcore.WriteSyncer {
	return noSync{
		Writer: w,
	}
}

type noSync struct {
	io.Writer
}

func (noSync) Sync() error {
	return nil
}
