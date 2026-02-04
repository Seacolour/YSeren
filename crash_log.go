package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

const crashLogFileName = "yseren-error.log"

func WriteCrashLog(title string, err error) {
	logPath := crashLogPath()
	if logPath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	f, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if openErr != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format(time.RFC3339)
	_, _ = fmt.Fprintf(f, "[%s] %s\n", ts, title)
	if err != nil {
		_, _ = fmt.Fprintf(f, "error: %v\n", err)
	}
	_, _ = fmt.Fprintf(f, "stack:\n%s\n\n", debug.Stack())
}

func crashLogPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), crashLogFileName)
}
