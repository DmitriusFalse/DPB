package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level int

const (
	LevelError Level = iota
	LevelDebug
)

type Logger struct {
	mu     sync.Mutex
	level  Level
	dir    string
	file   *os.File
	date   string
	stdout io.Writer
}

var defaultLogger = &Logger{level: LevelDebug, stdout: os.Stderr}

func Init(level Level, dir string) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
	defaultLogger.dir = dir
	defaultLogger.stdout = os.Stderr
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
		defaultLogger.file = nil
	}
	defaultLogger.openFile()
}

func (l *Logger) openFile() {
	if l.dir == "" {
		return
	}
	if err := os.MkdirAll(l.dir, 0755); err != nil {
		fmt.Fprintf(l.stdout, "[logger] mkdir %s: %v\n", l.dir, err)
		return
	}
	date := time.Now().Format("2006-01-02")
	path := filepath.Join(l.dir, "app-"+date+".log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(l.stdout, "[logger] open %s: %v\n", path, err)
		return
	}
	l.file = f
	l.date = date
}

func (l *Logger) rotate() {
	date := time.Now().Format("2006-01-02")
	if date != l.date {
		if l.file != nil {
			l.file.Close()
		}
		l.openFile()
	}
}

func (l *Logger) write(level string, format string, args ...interface{}) {
	l.rotate()
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] %s %s\n", time.Now().Format("15:04:05.000"), level, msg)
	if l.file != nil {
		l.file.WriteString(line)
	}
	l.stdout.Write([]byte(line))
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level > l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	switch level {
	case LevelError:
		l.write("ERROR", format, args...)
	case LevelDebug:
		l.write("DEBUG", format, args...)
	}
}

func Error(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
}

func Debug(format string, args ...interface{}) {
	defaultLogger.log(LevelDebug, format, args...)
}
