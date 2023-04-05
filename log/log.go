package log

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/decred/slog"
	"github.com/jrick/logrotate/rotator"
)

var mgmtLog = "mgmgt.log"

var (
	logRotator *rotator.Rotator
	backendLog = slog.NewBackend(logWriter{mgmtLog})
	Log        = backendLog.Logger("MGMGT")
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct {
	loggerID string
}

// Write writes the data in p to standard out and the log rotator.
func (l logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	return logRotator.Write(p)
}

func SetLogLevel(logLevel string) {
	level, _ := slog.LevelFromString(logLevel)
	Log.SetLevel(level)
}

func GetLogRotator() *rotator.Rotator {
	return logRotator
}

func InitLogRotator(logDir string) error {
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		return err
	}

	r, err := rotator.New(filepath.Join(logDir, mgmtLog), 32*1024, false, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		return err
	}

	logRotator = r
	return nil
}
