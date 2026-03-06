package logger

import (
	"cumt-nexus-api/internal/config"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.SugaredLogger

func InitLogger() error {
	c := config.Conf.Log
	appMode := config.Conf.App.Mode
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var encoder zapcore.Encoder
	if appMode == "debug" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   c.Filename,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   true,
	}
	var syncer zapcore.WriteSyncer
	if appMode == "debug" {
		syncer = zap.CombineWriteSyncers(zapcore.AddSync(lumberJackLogger), zapcore.AddSync(zapcore.Lock(os.Stdout)))
	} else {
		syncer = zapcore.AddSync(lumberJackLogger)
	}
	var level zapcore.Level
	switch c.Level {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	core := zapcore.NewCore(encoder, syncer, level)

	Log = zap.New(core, zap.AddCaller()).Sugar()

	return nil
}
