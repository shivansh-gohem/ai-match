package logger

import (
	"log"
	"os"
)

var (
	infoLog  *log.Logger
	errorLog *log.Logger
	fatalLog *log.Logger
)

func init() {
	infoLog = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	fatalLog = log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Info logs informational messages.
func Info(v ...interface{}) {
	infoLog.Println(v...)
}

// Infof logs formatted informational messages.
func Infof(format string, v ...interface{}) {
	infoLog.Printf(format, v...)
}

// Error logs error messages.
func Error(v ...interface{}) {
	errorLog.Println(v...)
}

// Errorf logs formatted error messages.
func Errorf(format string, v ...interface{}) {
	errorLog.Printf(format, v...)
}

// Fatal logs fatal messages and calls os.Exit(1).
func Fatal(v ...interface{}) {
	fatalLog.Fatal(v...)
}

// Fatalf logs formatted fatal messages and calls os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	fatalLog.Fatalf(format, v...)
}

// Printf exists for compatibility with existing log.Printf code
func Printf(format string, v ...interface{}) {
	infoLog.Printf(format, v...)
}

// Println exists for compatibility with existing log.Println code
func Println(v ...interface{}) {
	infoLog.Println(v...)
}
