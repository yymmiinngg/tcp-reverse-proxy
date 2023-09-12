package logger

import (
	"fmt"
	"strings"
	"time"
)

type Logger struct {
	Mode string
}

func (it *Logger) tag(tag string) string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t := time.Now().In(loc)
	return fmt.Sprintf("[%s] [%s] [%s]", t.Format("2006-01-02 15:04:05"), it.Mode, tag)
}

func (it *Logger) Info(message ...string) {
	fmt.Println(it.tag("INFO "), strings.Join(message, " "))
}

func (it *Logger) Error(err error, message ...string) {
	fmt.Println(it.tag("ERROR"), strings.Join(message, " ")+":", err)
}
