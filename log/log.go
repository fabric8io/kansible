package log

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	// IsDebugging toggles whether or not to enable debug output and behavior.
	IsDebugging = false

	// ErrorState denotes if application is in an error state.
	ErrorState = false
)

// Msg passes through the formatter, but otherwise prints exactly as-is.
//
// No prettification.
func Msg(format string, v ...interface{}) {
	fmt.Printf(appendNewLine(format), v...)
}

// Die prints an error and then call os.Exit(1).
func Die(format string, v ...interface{}) {
	Err(format, v...)
	if IsDebugging {
		panic(fmt.Sprintf(format, v...))
	}
	os.Exit(1)
}

// CleanExit prints a message and then exits with 0.
func CleanExit(format string, v ...interface{}) {
	Info(format, v...)
	os.Exit(0)
}

// Err prints an error message. It does not cause an exit.
func Err(format string, v ...interface{}) {
	fmt.Print(color.RedString("[ERROR] "))
	fmt.Printf(appendNewLine(format), v...)
	ErrorState = true
}

// Info prints a green-tinted message.
func Info(format string, v ...interface{}) {
	fmt.Print(color.GreenString("---> "))
	fmt.Printf(appendNewLine(format), v...)
}

// Debug prints a cyan-tinted message if IsDebugging is true.
func Debug(format string, v ...interface{}) {
	if IsDebugging {
		fmt.Print(color.CyanString("[DEBUG] "))
		fmt.Printf(appendNewLine(format), v...)
	}
}

// Warn prints a yellow-tinted warning message.
func Warn(format string, v ...interface{}) {
	fmt.Print(color.YellowString("[WARN] "))
	fmt.Printf(appendNewLine(format), v...)
}

func appendNewLine(format string) string {
	return format + "\n"
}
