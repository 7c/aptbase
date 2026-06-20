// Package debug provides a process-global, stderr-based debug logger.
//
// It is a leaf package (no imports of client/config/cmd) so any layer can emit
// debug lines via Logf without plumbing a logger through call sites. Enabling is
// a single switch: set Enabled = true (done by the --debug flag / APTBASE_DEBUG
// / config). All output goes to stderr so it never pollutes --json on stdout.
package debug

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/fatih/color"
)

// Enabled turns debug output on. When false, all logging is a no-op.
var Enabled bool

var (
	tagColor  = color.New(color.FgMagenta, color.Faint)
	headColor = color.New(color.FgMagenta, color.Bold)
)

// now is overridable in tests; defaults to wall-clock time.
var now = time.Now

// Logf writes a timestamped debug line to stderr when Enabled.
func Logf(format string, args ...any) {
	if !Enabled {
		return
	}
	tag := tagColor.Sprintf("[debug %s]", now().Format("15:04:05.000"))
	fmt.Fprintf(os.Stderr, "%s %s\n", tag, fmt.Sprintf(format, args...))
}

// Section writes a bold debug header to group related output.
func Section(title string) {
	if !Enabled {
		return
	}
	fmt.Fprintln(os.Stderr, headColor.Sprintf("[debug] === %s ===", title))
}

// secretField matches JSON string values for sensitive keys so they can be
// masked before logging request bodies.
var secretField = regexp.MustCompile(`("(?:[Pp]assword|[Pp]assphrase|GpgKeyArmor)"\s*:\s*")(?:\\.|[^"\\])*(")`)

// Redact masks the values of known sensitive JSON fields in b, returning a
// string safe to log.
func Redact(b []byte) string {
	return secretField.ReplaceAllString(string(b), `${1}***${2}`)
}
