package utils

import (
	"fmt"
	"log"
	"os"
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
	l.print("INFO  - " + l.prefix + fmt.Sprintln(a...))
}
func (l *CustomLogger) InfoLnf(format string, a ...any) {
	l.Println("INFO  - " + l.prefix + fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Infof(format string, a ...any) {
	l.Printf("INFO  - "+l.prefix+format, a...)
}

func (l *CustomLogger) WarnLn(a ...any) {
	l.print("WARN  - " + l.prefix + fmt.Sprintln(a...))
}
func (l *CustomLogger) WarnLnf(format string, a ...any) {
	l.Println("WARN  - " + l.prefix + fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Warnf(format string, a ...any) {
	l.Printf("WARN  - "+l.prefix+format, a...)
}

func (l *CustomLogger) DebugLn(a ...any) {
	l.print("DEBUG - " + l.prefix + fmt.Sprintln(a...))
}
func (l *CustomLogger) DebugLnf(format string, a ...any) {
	l.Println("DEBUG - " + l.prefix + fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Debugf(format string, a ...any) {
	l.Printf("DEBUG - "+l.prefix+format, a...)
}

func (l *CustomLogger) ErrorLn(a ...any) {
	l.print("ERROR - " + l.prefix + fmt.Sprintln(a...))
}
func (l *CustomLogger) ErrorLnf(format string, a ...any) {
	l.Println("ERROR - " + l.prefix + fmt.Sprintf(format, a...))
}
func (l *CustomLogger) Errorf(format string, a ...any) {
	l.Printf("ERROR - "+l.prefix+format, a...)
}

func (l *CustomLogger) FatalLn(a ...any) {
	l.print("FATAL - " + l.prefix + fmt.Sprintln(a...))
	// TODO close log writer
	os.Exit(1)
}
func (l *CustomLogger) FatalLnf(format string, a ...any) {
	l.Println("FATAL - " + l.prefix + fmt.Sprintf(format, a...))
	os.Exit(1)
}
func (l *CustomLogger) Fatalf(format string, a ...any) {
	l.Printf("FATAL - "+l.prefix+format, a...)
	os.Exit(1)
}

func (l *CustomLogger) print(s string) {
	l.printFn(s)
}

func (l *CustomLogger) SetPrefix(prefix string) {
	if prefix != "" {
		prefix = "[" + prefix + "]"
		l.prefix = fmt.Sprintf("%-16s ", prefix)
	} else {
		l.prefix = ""
	}
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
	logger.SetPrefix(prefix)
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

var parsingLogger = NewLogger("Parsing")
