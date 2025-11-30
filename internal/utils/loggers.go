package utils

import (
	"fmt"
	"log"
)

type CustomLogger struct {
	printFn func(string)
}

func (l *CustomLogger) Println(a ...any) {
	l.printFn(fmt.Sprintln(a...))
}

func (l *CustomLogger) Printf(format string, a ...any) {
	l.printFn(fmt.Sprintf(format, a...))
}

func (l *CustomLogger) InfoLn(a ...any) {
	l.printFn("[Info] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) InfoLnf(format string, a ...any) {
	l.Println("[Info]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Infof(format string, a ...any) {
	l.Printf("[Info] "+format, a...)
}

func (l *CustomLogger) WarnLn(a ...any) {
	l.printFn("[Warning] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) WarnLnf(format string, a ...any) {
	l.Println("[Warning]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Warnf(format string, a ...any) {
	l.Printf("[Warning] "+format, a...)
}

func (l *CustomLogger) DebugLn(a ...any) {
	l.printFn("[Debug] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) DebugLnf(format string, a ...any) {
	l.Println("[Debug]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Debugf(format string, a ...any) {
	l.Printf("[Debug] "+format, a...)
}

func (l *CustomLogger) ErrorLn(a ...any) {
	l.printFn("[Error] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) ErrorLnf(format string, a ...any) {
	l.Println("[Error]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Errorf(format string, a ...any) {
	l.Printf("[Error] "+format, a...)
}

type Printable interface {
	Print(string)
}

func (l *CustomLogger) Bind(p Printable) {
	l.printFn = p.Print
}

func NewLogger() *CustomLogger {
	return &CustomLogger{printFn: func(s string) {
		log.Print(s)
	}}
}

type UniqueLogger struct {
	CustomLogger
	lastLog string
}

func NewUniqueLogger() *UniqueLogger {
	logger := &UniqueLogger{}
	logger.Bind(logger)
	return logger
}

func (l *UniqueLogger) Print(str string) {
	if str == l.lastLog {
		return
	}
	l.lastLog = str
	log.Print(str)
}
