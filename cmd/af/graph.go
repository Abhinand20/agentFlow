package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Abhinand20/agentFlow/internal/dot"
)

func cmdGraph(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// DOT is target-neutral; the flag is accepted for forward compatibility
	// and symmetry with `af build`.
	_ = fs.String("target", "", "host target (accepted for forward compatibility; DOT is target-neutral)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af graph <file> [--target claude-code|cursor]")
		fs.PrintDefaults()
	}
	operands, err := parseArgs(fs, args)
	if err != nil {
		return 2
	}
	if len(operands) != 1 {
		fs.Usage()
		return 2
	}

	res, hasErrors := compileOrReport(operands[0], stderr)
	if hasErrors {
		return 1
	}
	fmt.Fprint(stdout, dot.Emit(res.IR))
	return 0
}
