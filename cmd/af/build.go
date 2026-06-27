package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/manifest"
)

func cmdBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "host target ("+availableTargets()+")")
	out := fs.String("out", ".", "output directory")
	emitIR := fs.Bool("emit-ir", false, "print IR JSON to stdout instead of writing host files (ignores --target)")
	noManifest := fs.Bool("no-manifest", false, "skip writing the build manifest")
	prune := fs.Bool("prune", false, "remove artifacts dropped since the previous build of this source")
	force := fs.Bool("force", false, "with --prune, delete modified artifacts")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af build <file> --target <host> [--out dir] [--emit-ir] [--no-manifest] [--prune] [--force]")
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
	sourcePath := operands[0]

	res, hasErrors := compileOrReport(sourcePath, stderr)
	if hasErrors {
		return 1
	}

	if *emitIR {
		if *target != "" {
			fmt.Fprintf(stderr, "build: --emit-ir ignores --target (%s)\n", availableTargets())
		}
		data, err := ir.Marshal(res.IR)
		if err != nil {
			fmt.Fprintf(stderr, "build: encode IR: %v\n", err)
			return 1
		}
		if _, err := fmt.Fprintln(stdout, string(data)); err != nil {
			fmt.Fprintf(stderr, "build: write IR: %v\n", err)
			return 1
		}
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

	var prior *manifest.Manifest
	var current *manifest.Manifest
	if !*noManifest {
		priorLoaded, found, err := manifest.Load(*out, *target, sourcePath)
		if err != nil {
			fmt.Fprintf(stderr, "build: load manifest: %v\n", err)
			return 1
		}
		if found {
			prior = priorLoaded
		}

		irData, err := ir.Marshal(res.IR)
		if err != nil {
			fmt.Fprintf(stderr, "build: encode IR for manifest: %v\n", err)
			return 1
		}
		m := manifest.Build(manifest.BuildOptions{
			Target:       *target,
			SourcePath:   sourcePath,
			SourceSHA256: manifest.HashSource(res.Source),
			IRHash:       manifest.HashBytes(irData),
			FS:           hostFS,
			Prior:        prior,
			ToolVersion:  toolVersion,
		})
		current = &m

		all, err := manifest.LoadAll(*out, *target, func(msg string) {
			fmt.Fprintln(stderr, msg)
		})
		if err != nil {
			fmt.Fprintf(stderr, "build: load manifests: %v\n", err)
			return 1
		}
		for path, detail := range manifest.Overlaps(current, all) {
			fmt.Fprintf(stderr, "warning AF312: artifact %s overlaps another source (%s)\n", path, detail)
		}

		data, err := manifest.Marshal(m)
		if err != nil {
			fmt.Fprintf(stderr, "build: encode manifest: %v\n", err)
			return 1
		}
		hostFS.Write(manifest.ManifestRelPath(*target, sourcePath), data)
	}

	if err := hostFS.Flush(*out); err != nil {
		fmt.Fprintf(stderr, "build: %v\n", err)
		return 1
	}
	for _, p := range hostFS.Paths() {
		fmt.Fprintln(stdout, filepath.Join(*out, filepath.FromSlash(p)))
	}

	if *prune && prior != nil && current != nil && !*noManifest {
		if err := pruneArtifacts(*out, prior, current, *force, stderr); err != nil {
			fmt.Fprintf(stderr, "build: prune: %v\n", err)
			return 1
		}
	}

	return 0
}

func pruneArtifacts(out string, prior, current *manifest.Manifest, force bool, stderr io.Writer) error {
	changes := manifest.Diff(prior.CurrentRecord(), current.CurrentRecord())
	report := manifest.DriftCheck(prior, out)
	modified := make(map[string]struct{}, len(report.Modified))
	for _, art := range report.Modified {
		modified[art.Path] = struct{}{}
	}

	for _, rel := range changes.Removed {
		if _, ok := modified[rel]; ok && !force {
			fmt.Fprintf(stderr, "warning AF310: skip pruning modified artifact %s (use --force)\n", rel)
			continue
		}
		full := filepath.Join(out, filepath.FromSlash(rel))
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Fprintf(stderr, "build: pruned %s\n", rel)
	}
	return nil
}

func availableTargets() string {
	names := binding.Names()
	if len(names) == 0 {
		return "no targets registered"
	}
	return "available: " + strings.Join(names, ", ")
}
