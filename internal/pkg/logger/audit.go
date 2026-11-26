package logger

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var audit = newAudit(os.Stdout)

// newAudit builds a logger that always writes JSON at info level, so the
// configured log level or format cannot drop or reshape audit records.
func newAudit(w io.Writer) *zap.Logger {
	enc := zap.NewProductionEncoderConfig()
	enc.TimeKey = "timestamp"
	enc.MessageKey = "message"
	enc.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(zapcore.NewJSONEncoder(enc), zapcore.Lock(zapcore.AddSync(w)), zapcore.InfoLevel)
	return zap.New(core).With(zap.Bool("audit", true))
}

// Audit writes one audit record to stdout.
func Audit(action string, fields ...zap.Field) {
	audit.Info(action, fields...)
}
