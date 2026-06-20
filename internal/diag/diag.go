package diag

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// Severity classifies a diagnostic.
type Severity int

const (
	Error Severity = iota
	Warning
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "unknown"
	}
}

// Diagnostic is a single compiler message with source location.
type Diagnostic struct {
	Code     string
	Severity Severity
	Msg      string
	Pos      lexer.Position
}

// Diagnostics is an ordered collection of diagnostics.
type Diagnostics []Diagnostic

// HasErrors reports whether any diagnostic is an error.
func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Severity == Error {
			return true
		}
	}
	return false
}

// Add appends diagnostics to the collection.
func (d *Diagnostics) Add(diags ...Diagnostic) {
	*d = append(*d, diags...)
}

// Render formats diagnostics for human-readable CLI output.
// Each diagnostic prints file:line:col, severity, code, message, the source line,
// and a caret at the column when source is available.
func Render(source string, diags Diagnostics) string {
	if len(diags) == 0 {
		return ""
	}

	lines := strings.Split(source, "\n")
	var b strings.Builder

	for i, d := range diags {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s:%d:%d: %s %s: %s",
			d.Pos.Filename, d.Pos.Line, d.Pos.Column,
			d.Severity, d.Code, d.Msg)

		if d.Pos.Line <= 0 || d.Pos.Line > len(lines) {
			continue
		}

		line := lines[d.Pos.Line-1]
		b.WriteByte('\n')
		b.WriteString(line)
		b.WriteByte('\n')

		col := d.Pos.Column
		if col < 1 {
			col = 1
		}
		b.WriteString(strings.Repeat(" ", col-1))
		b.WriteByte('^')
	}

	return b.String()
}
