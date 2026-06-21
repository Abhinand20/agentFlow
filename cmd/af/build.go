package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/ir"
)

func cmdBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "host target ("+availableTargets()+")")
	out := fs.String("out", ".", "output directory")
	emitIR := fs.Bool("emit-ir", false, "print IR JSON to stdout instead of writing host files")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af build <file> --target <host> [--out dir] [--emit-ir]")
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

	if *emitIR {
		data, err := ir.Marshal(res.IR)
		if err != nil {
			fmt.Fprintf(stderr, "build: encode IR: %v\n", err)
			return 1
		}
		stdout.Write(data)
		fmt.Fprintln(stdout)
		return 0
	}

	if *target == "" {
		fmt.Fprintf(stderr, "build: --target is required (%s)\n", availableTargets())
		return 2
	}
	b, ok := binding.Get(*target)
	if !ok {
		fmt.Fprintf(stderr, "build: unknown target %q (%s)\n", *target, availableTargets())
		return 2
	}

	hostFS, bdiags := b.Emit(res.IR)
	emitDiags(stderr, res.Source, bdiags)
	if bdiags.HasErrors() {
		return 1
	}

	if err := hostFS.Flush(*out); err != nil {
		fmt.Fprintf(stderr, "build: %v\n", err)
		return 1
	}
	for _, p := range hostFS.Paths() {
		fmt.Fprintln(stdout, filepath.Join(*out, filepath.FromSlash(p)))
	}
	return 0
}

func availableTargets() string {
	names := binding.Names()
	if len(names) == 0 {
		return "no targets registered"
	}
	return "available: " + strings.Join(names, ", ")
}
