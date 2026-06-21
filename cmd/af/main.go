package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/pipeline"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches a subcommand and returns the process exit code.
//
// Exit policy (consistent across commands; CI and users depend on it):
//   - 0: success (clean, or warnings only)
//   - 1: compile/build errors (diagnostics with error severity)
//   - 2: usage errors (bad flags, missing/unknown arguments, unknown command)
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usageText())
		return 0
	}

	switch args[0] {
	case "validate":
		return cmdValidate(args[1:], stdout, stderr)
	case "graph":
		return cmdGraph(args[1:], stdout, stderr)
	case "build":
		return cmdBuild(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usageText())
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usageText())
		return 2
	}
}

// parseArgs parses fs allowing flags and positional operands to be
// interleaved, e.g. `af build review.af --target cursor`. The stdlib flag
// package stops at the first non-flag argument; this re-parses the remainder
// after each operand so the natural file-first invocation works.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var operands []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return operands, nil
		}
		operands = append(operands, rest[0])
		args = rest[1:]
	}
}

// emitDiags renders diagnostics to stderr. Diagnostics carrying a source
// position (parse/resolve/validate) go through diag.Render for the file:line
// caret view; positionless diagnostics (e.g. AF3xx binding negotiation) print
// as "<severity> <code>: <msg>" so the noise-free location is omitted.
func emitDiags(stderr io.Writer, source string, diags diag.Diagnostics) {
	var positioned diag.Diagnostics
	for _, d := range diags {
		if !d.HasSourceLocation() {
			fmt.Fprintf(stderr, "%s %s: %s\n", d.Severity, d.Code, d.Msg)
			continue
		}
		positioned.Add(d)
	}
	if len(positioned) > 0 {
		fmt.Fprintln(stderr, diag.Render(source, positioned))
	}
}

// compileOrReport compiles path, printing any diagnostics to stderr. It reports
// whether compilation produced errors so callers can stop with exit code 1.
func compileOrReport(path string, stderr io.Writer) (pipeline.Result, bool) {
	res, diags := pipeline.Compile(path)
	emitDiags(stderr, res.Source, diags)
	return res, diags.HasErrors()
}

func usageText() string {
	return `af — AgentFlow compiler

Usage:
  af <command> [arguments]

Commands:
  validate   Check an .af file for errors
  build      Compile an .af file to host configuration
  graph      Print the resolved flow graph as DOT

Run "af <command> -h" for command-specific flags.
`
}
