package logger

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Logger struct {
	mode  string
	out   loggerOut
	debug bool
}

type loggerOut interface {
	Out(msg string)
	Close()
}

func MakeLogger(mode, out string, debug bool) (*Logger, error) {
	if out == "console" {
		return &Logger{mode: mode, debug: debug, out: &loggerOut2Console{}}, nil
	}
	file, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{mode: mode, debug: debug, out: &loggerOut2File{file: file}}, nil
}

type loggerOut2Console struct{}

func (it *loggerOut2Console) Out(msg string) {
	fmt.Println(msg)
}

func (it *loggerOut2Console) Close() {
}

type loggerOut2File struct {
	file *os.File
}

func (it *loggerOut2File) Close() {
	it.file.Close()
}

func (it *loggerOut2File) Out(msg string) {
	it.file.WriteString(msg + "\n")
}

func (it *Logger) tag(tag string) string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t := time.Now().In(loc)
	return fmt.Sprintf("[%s] [%s] [%s]", t.Format("2006-01-02 15:04:05"), it.mode, tag)
}

func (it *Logger) Info(message ...string) {
	it.output(it.tag("INFO "), strings.Join(message, " "))
}

func (it *Logger) Error(err error, message ...string) {
	it.output(it.tag("ERROR"), strings.Join(message, " ")+":", err.Error())
}

func (it *Logger) Debug(message ...string) {
	if it.debug {
		it.output(it.tag("DEBUG"), strings.Join(message, " "))
	}
}

func (it *Logger) output(msg ...string) {
	it.out.Out(strings.Join(msg, " "))
}

func (it *Logger) Close() {
	it.out.Close()
}
