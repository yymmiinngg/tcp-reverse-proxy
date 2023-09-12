package logger

import (
	"fmt"
	"strings"
	"time"
)

var Type = "*"

func tag(tag string) string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t := time.Now().In(loc)
	return fmt.Sprintf("[%s] [%s] [%s]", t.Format("2006-01-02 15:04:05"), Type, tag)
}

func Info(message ...string) {
	fmt.Println(tag("INFO "), strings.Join(message, " "))
}

func Error(err error, message ...string) {
	fmt.Println(tag("ERROR"), strings.Join(message, " ")+":", err)
}
