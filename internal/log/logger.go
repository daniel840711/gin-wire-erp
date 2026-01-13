package log

import (
	"fmt"
	"os"

	"interchange/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(conf *config.Configuration) (*zap.Logger, error) {
	// 1) 解析最小輸出層級（作為全域門檻）
	var lvl zapcore.Level
	switch conf.Log.Level {
	case "debug":
		lvl = zap.DebugLevel
	case "info":
		lvl = zap.InfoLevel
	case "warn":
		lvl = zap.WarnLevel
	case "error":
		lvl = zap.ErrorLevel
	case "dpanic":
		lvl = zap.DPanicLevel
	case "panic":
		lvl = zap.PanicLevel
	case "fatal":
		lvl = zap.FatalLevel
	default:
		lvl = zap.InfoLevel
	}

	atomic := zap.NewAtomicLevelAt(lvl)

	// 2) Encoder 設定（JSON、ISO8601 時間、caller/level 鍵等）
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.MessageKey = "message"
	encCfg.LevelKey = "level"
	encCfg.TimeKey = "ts"
	encCfg.CallerKey = "caller"
	encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	encCfg.EncodeCaller = zapcore.ShortCallerEncoder
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewJSONEncoder(encCfg)

	// 3) 分流到 stdout / stderr（同時受全域門檻控制）
	stdoutWriter := zapcore.AddSync(os.Stdout)
	stderrWriter := zapcore.AddSync(os.Stderr)

	stdoutLevel := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return atomic.Enabled(l) && l < zapcore.WarnLevel
	})
	stderrLevel := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return atomic.Enabled(l) && l >= zapcore.WarnLevel
	})

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stdoutWriter, stdoutLevel),
		zapcore.NewCore(encoder, stderrWriter, stderrLevel),
	)

	// 4) Options：顯示 caller；stacktrace 只在 Error+ 時出現
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	}

	logger := zap.New(core, opts...)
	logger.Info(fmt.Sprintf("zap logger set level: %s", conf.Log.Level))

	return logger, nil
}
