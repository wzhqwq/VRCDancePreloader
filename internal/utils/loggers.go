package utils

import (
	"fmt"
	"log"
)

type CustomLogger struct {
	printFn func(string)
	prefix  string
}

func (l *CustomLogger) Println(a ...any) {
	l.print(fmt.Sprintln(a...))
}

func (l *CustomLogger) Printf(format string, a ...any) {
	l.print(fmt.Sprintf(format, a...))
}

func (l *CustomLogger) InfoLn(a ...any) {
	l.print("[Info] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) InfoLnf(format string, a ...any) {
	l.Println("[Info]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Infof(format string, a ...any) {
	l.Printf("[Info] "+format, a...)
}

func (l *CustomLogger) WarnLn(a ...any) {
	l.print("[Warning] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) WarnLnf(format string, a ...any) {
	l.Println("[Warning]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Warnf(format string, a ...any) {
	l.Printf("[Warning] "+format, a...)
}

func (l *CustomLogger) DebugLn(a ...any) {
	l.print("[Debug] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) DebugLnf(format string, a ...any) {
	l.Println("[Debug]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Debugf(format string, a ...any) {
	l.Printf("[Debug] "+format, a...)
}

func (l *CustomLogger) ErrorLn(a ...any) {
	l.print("[Error] " + fmt.Sprintln(a...))
}
func (l *CustomLogger) ErrorLnf(format string, a ...any) {
	l.Println("[Error]", fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Errorf(format string, a ...any) {
	l.Printf("[Error] "+format, a...)
}

func (l *CustomLogger) print(s string) {
	l.printFn(l.prefix + s)
}

func (l *CustomLogger) SetPrefix(prefix string) {
	l.prefix = prefix
}

type Printable interface {
	Print(string)
}

func (l *CustomLogger) Bind(p Printable) {
	l.printFn = p.Print
}

func NewLogger(prefix string) *CustomLogger {
	logger := &CustomLogger{
		printFn: func(s string) {
			log.Print(s)
		},
	}
	logger.prefix = prefix
	return logger
}

type UniqueLogger struct {
	CustomLogger
	lastLog string
}

func NewUniqueLogger(prefix string) *UniqueLogger {
	logger := &UniqueLogger{}
	logger.Bind(logger)
	logger.SetPrefix(prefix)
	return logger
}

func (l *UniqueLogger) Print(str string) {
	if str == l.lastLog {
		return
	}
	l.lastLog = str
	log.Print(str)
}
