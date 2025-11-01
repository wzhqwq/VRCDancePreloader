package utils

import (
	"fmt"
	"log"
)

type UniqueLogger struct {
	lastLog string
}

func NewUniqueLogger() *UniqueLogger {
	return &UniqueLogger{}
}

func (l *UniqueLogger) Print(str string) {
	if str == l.lastLog {
		return
	}
	l.lastLog = str
	log.Print(str)
}

func (l *UniqueLogger) Println(a ...any) {
	l.Print(fmt.Sprintln(a...))
}

func (l *UniqueLogger) Printf(format string, a ...any) {
	l.Print(fmt.Sprintf(format, a...))
}

func (l *UniqueLogger) InfoLn(a ...any) {
	l.Print("[Info] " + fmt.Sprintln(a...))
}
func (l *UniqueLogger) InfoLnf(format string, a ...any) {
	l.Println("[Info]", fmt.Sprintf(format, a...))
}
func (l *UniqueLogger) Infof(format string, a ...any) {
	l.Printf("[Info] "+format, a...)
}

func (l *UniqueLogger) WarnLn(a ...any) {
	l.Print("[Warning] " + fmt.Sprintln(a...))
}
func (l *UniqueLogger) WarnLnf(format string, a ...any) {
	l.Println("[Warning]", fmt.Sprintf(format, a...))
}
func (l *UniqueLogger) Warnf(format string, a ...any) {
	l.Printf("[Warning] "+format, a...)
}

func (l *UniqueLogger) DebugLn(a ...any) {
	l.Print("[Debug] " + fmt.Sprintln(a...))
}
func (l *UniqueLogger) DebugLnf(format string, a ...any) {
	l.Println("[Debug]", fmt.Sprintf(format, a...))
}
func (l *UniqueLogger) Debugf(format string, a ...any) {
	l.Printf("[Debug] "+format, a...)
}

func (l *UniqueLogger) ErrorLn(a ...any) {
	l.Print("[Error] " + fmt.Sprintln(a...))
}
func (l *UniqueLogger) ErrorLnf(format string, a ...any) {
	l.Println("[Error]", fmt.Sprintf(format, a...))
}
func (l *UniqueLogger) Errorf(format string, a ...any) {
	l.Printf("[Error] "+format, a...)
}
