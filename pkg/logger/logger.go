package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// InitGlobalLogger initializes zap global logger
func InitGlobalLogger() {
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	logger := zap.New(zapcore.NewCore(encoder, getLogWriter(), zapcore.DebugLevel))
	zap.ReplaceGlobals(logger)
}

func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename: "./manager.log",
	}
	return zapcore.AddSync(lumberJackLogger)
}
