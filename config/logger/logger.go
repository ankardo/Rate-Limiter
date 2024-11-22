package logger

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func init() {
	logConfiguration := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			TimeKey:      "time",
			EncodeLevel:  zapcore.LowercaseLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	log, _ = logConfiguration.Build()
}

func GetZapLogger() *zap.Logger {
	return log
}

type ZapWriter struct {
	logger *zap.Logger
}

func (z *ZapWriter) Write(p []byte) (n int, err error) {
	z.logger.Info(string(p))
	return len(p), nil
}

func GetZapWriter() io.Writer {
	return &ZapWriter{logger: log}
}

func Debug(message string, tags ...zap.Field) {
	log.Debug(message, tags...)
	log.Sync()
}

func Info(message string, tags ...zap.Field) {
	log.Info(message, tags...)
	log.Sync()
}

func Error(message string, err error, tags ...zap.Field) {
	if err != nil {
		tags = append(tags, zap.NamedError("error", err))
	}
	log.Error(message, tags...)
	log.Sync()
}
