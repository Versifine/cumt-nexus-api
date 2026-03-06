package logger

import (
	"cumt-nexus-api/internal/config"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.SugaredLogger

func init() {
	// 兜底日志器：保证在 InitLogger 之前也可以安全打印日志（例如配置初始化失败时）
	base, err := zap.NewDevelopment()
	if err != nil {
		Log = zap.NewNop().Sugar()
		return
	}
	Log = base.Sugar()
}

// Sync 统一处理日志刷盘，避免在不同平台出现重复样板代码
func Sync() {
	if Log == nil {
		return
	}
	_ = Log.Sync()
}

func InitLogger() error {
	c := config.Conf.Log
	appMode := config.Conf.App.Mode

	if dir := filepath.Dir(c.Filename); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

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
	switch strings.ToLower(c.Level) {
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
