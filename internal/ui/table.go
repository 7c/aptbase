package ui

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
)

// colGap is the spacing between table columns.
const colGap = "  "

var headerStyle = color.New(color.Bold, color.FgCyan)

// Table renders rows aligned in columns with a colored header and a separator
// rule. Column widths are computed from the visible (rune) length of cells, so
// colored cells do not break alignment — unlike text/tabwriter, which counts
// ANSI escape bytes.
func Table(headers []string, rows [][]string) {
	cols := len(headers)
	if cols == 0 {
		return
	}
	widths := make([]int, cols)
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i := 0; i < cols && i < len(row); i++ {
			if w := utf8.RuneCountInString(row[i]); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Header.
	var header strings.Builder
	for i, h := range headers {
		header.WriteString(headerStyle.Sprint(padRight(h, widths[i])))
		if i < cols-1 {
			header.WriteString(colGap)
		}
	}
	fmt.Fprintln(os.Stdout, strings.TrimRight(header.String(), " "))

	// Separator rule.
	var rule strings.Builder
	for i := range headers {
		rule.WriteString(strings.Repeat("─", widths[i]))
		if i < cols-1 {
			rule.WriteString(colGap)
		}
	}
	dimColor.Fprintln(os.Stdout, rule.String())

	// Rows.
	for _, row := range rows {
		var line strings.Builder
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line.WriteString(padRight(cell, widths[i]))
			if i < cols-1 {
				line.WriteString(colGap)
			}
		}
		fmt.Fprintln(os.Stdout, strings.TrimRight(line.String(), " "))
	}
}

// KeyValues prints aligned key/value pairs (for "show" style output).
func KeyValues(pairs [][2]string) {
	width := 0
	for _, p := range pairs {
		if w := utf8.RuneCountInString(p[0]); w > width {
			width = w
		}
	}
	keyStyle := color.New(color.Bold)
	for _, p := range pairs {
		fmt.Fprintf(os.Stdout, "%s  %s\n", keyStyle.Sprint(padRight(p[0], width)), p[1])
	}
}

// padRight pads s with spaces to the given visible (rune) width.
func padRight(s string, width int) string {
	n := width - utf8.RuneCountInString(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}
