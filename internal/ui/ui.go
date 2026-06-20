// Package ui provides small helpers for colorful, consistent terminal output.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warnColor    = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgCyan)
	dimColor     = color.New(color.Faint)
)

// Success prints a green success line prefixed with a check mark.
func Success(format string, a ...any) {
	successColor.Fprintf(os.Stdout, "✓ "+format+"\n", a...)
}

// Info prints an informational line in cyan.
func Info(format string, a ...any) {
	infoColor.Fprintf(os.Stdout, format+"\n", a...)
}

// Warn prints a warning line in yellow to stderr.
func Warn(format string, a ...any) {
	warnColor.Fprintf(os.Stderr, "! "+format+"\n", a...)
}

// Error prints an error line in red to stderr.
func Error(format string, a ...any) {
	errorColor.Fprintf(os.Stderr, "✗ "+format+"\n", a...)
}

// Dim prints faint, secondary text.
func Dim(format string, a ...any) {
	dimColor.Fprintf(os.Stdout, format+"\n", a...)
}

// Heading prints a bold underlined section heading.
func Heading(text string) {
	fmt.Fprintln(os.Stdout)
	color.New(color.Bold, color.Underline).Fprintln(os.Stdout, text)
}
