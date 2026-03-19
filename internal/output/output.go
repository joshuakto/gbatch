package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Colors returns false if NO_COLOR is set or TERM=dumb.
func Colors() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return true
}

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
)

func color(c, s string) string {
	if !Colors() {
		return s
	}
	return c + s + Reset
}

func Green_(s string) string  { return color(Green, s) }
func Red_(s string) string    { return color(Red, s) }
func Yellow_(s string) string { return color(Yellow, s) }
func Blue_(s string) string   { return color(Blue, s) }
func Cyan_(s string) string   { return color(Cyan, s) }
func Bold_(s string) string   { return color(Bold, s) }
func Dim_(s string) string    { return color(Dim, s) }

// Status prefixes with unicode fallback.
func SuccessPrefix() string { return Green_("✓") }
func ErrorPrefix() string   { return Red_("✗") }
func WarnPrefix() string    { return Yellow_("⚠") }

// Success prints a success message: ✓ message
func Success(msg string) {
	fmt.Printf("%s %s\n", SuccessPrefix(), msg)
}

// Error prints an error message: ✗ message
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", ErrorPrefix(), msg)
}

// ErrorHint prints an error with an actionable hint.
func ErrorHint(msg, hint string) {
	fmt.Fprintf(os.Stderr, "%s %s\n  Hint: %s\n", ErrorPrefix(), msg, hint)
}

// Warn prints a warning: ⚠ message
func Warn(msg string) {
	fmt.Printf("%s %s\n", WarnPrefix(), msg)
}

// Info prints an informational message (cyan).
func Info(msg string) {
	fmt.Println(Cyan_(msg))
}

// JSON prints v as indented JSON to stdout.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// StatusColor returns the colored status string based on job state.
func StatusColor(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCEEDED", "DONE":
		return Green_(status)
	case "FAILED":
		return Red_(status)
	case "RUNNING":
		return Cyan_(status)
	case "PENDING", "QUEUED", "SCHEDULED":
		return Yellow_(status)
	default:
		return status
	}
}

// Table prints aligned columns. widths specifies min width per column.
func Table(headers []string, rows [][]string, widths []int) {
	// Calculate actual widths
	for i, h := range headers {
		if i < len(widths) && len(h) > widths[i] {
			widths[i] = len(h)
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		w := 10
		if i < len(widths) {
			w = widths[i]
		}
		fmt.Printf("%-*s  ", w, Bold_(h))
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		w := 10
		if i < len(widths) {
			w = widths[i]
		}
		fmt.Printf("%-*s  ", w, Dim_(strings.Repeat("-", w)))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			w := 10
			if i < len(widths) {
				w = widths[i]
			}
			fmt.Printf("%-*s  ", w, cell)
		}
		fmt.Println()
	}
}
