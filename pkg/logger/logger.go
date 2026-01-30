package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "ts"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	var err error
	L, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

// WithComponent 回傳帶有 component 欄位的 logger，供 MQ、handler、service 等使用
func WithComponent(component string) *zap.Logger {
	return L.With(zap.String("component", component))
}
