package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/manifest"
)

func cmdClean(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "host target ("+availableTargets()+")")
	out := fs.String("out", ".", "output directory")
	all := fs.Bool("all", false, "clean artifacts for every tracked source")
	dryRun := fs.Bool("dry-run", false, "print deletions without removing files")
	force := fs.Bool("force", false, "delete modified or cross-source-owned artifacts")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af clean <file> --target <host> [--out dir] [--all] [--dry-run] [--force]")
		fs.PrintDefaults()
	}
	operands, err := parseArgs(fs, args)
	if err != nil {
		return 2
	}

	if *target == "" {
		fmt.Fprintf(stderr, "clean: --target is required (%s)\n", availableTargets())
		return 2
	}
	if _, ok := binding.Get(*target); !ok {
		fmt.Fprintf(stderr, "clean: unknown target %q (%s)\n", *target, availableTargets())
		return 2
	}

	var sourcePath string
	switch {
	case *all && len(operands) > 0:
		fmt.Fprintln(stderr, "clean: specify either a source file or --all, not both")
		return 2
	case *all:
	case len(operands) == 1:
		sourcePath = operands[0]
	case len(operands) == 0:
		fmt.Fprintln(stderr, "clean: specify an .af source or --all")
		fs.Usage()
		return 2
	default:
		fs.Usage()
		return 2
	}

	warn := func(msg string) { fmt.Fprintln(stderr, msg) }
	allManifests, err := manifest.LoadAll(*out, *target, warn)
	if err != nil {
		fmt.Fprintf(stderr, "clean: %v\n", err)
		return 1
	}

	var manifests []*manifest.Manifest
	if *all {
		if len(allManifests) == 0 {
			fmt.Fprintf(stderr, "clean: no manifests found under %s\n", manifest.ManifestsDir(*target))
			return 1
		}
		manifests = allManifests
	} else {
		m, ok, loadErr := manifest.Load(*out, *target, sourcePath)
		if loadErr != nil {
			fmt.Fprintf(stderr, "clean: %v\n", loadErr)
			return 1
		}
		if !ok {
			fmt.Fprintf(stderr, "clean: no manifest found for %s; nothing to clean\n", sourcePath)
			return 1
		}
		manifests = []*manifest.Manifest{m}
	}

	owners := manifest.ArtifactOwners(allManifests)

	var toDelete []string
	var manifestFiles []string
	for _, m := range manifests {
		manifestFiles = append(manifestFiles, manifest.ManifestRelPath(*target, m.Source.Path))
		report := manifest.DriftCheck(m, *out)
		for _, art := range m.Artifacts {
			if shouldSkipClean(art, m, owners, report, *force, stderr) {
				continue
			}
			toDelete = append(toDelete, art.Path)
		}
	}

	toDelete = uniqueSorted(toDelete)
	manifestFiles = uniqueSorted(manifestFiles)

	if *dryRun {
		for _, p := range toDelete {
			fmt.Fprintln(stdout, filepath.Join(*out, filepath.FromSlash(p)))
		}
		for _, p := range manifestFiles {
			fmt.Fprintln(stdout, filepath.Join(*out, filepath.FromSlash(p)))
		}
		return 0
	}

	for _, rel := range toDelete {
		full := filepath.Join(*out, filepath.FromSlash(rel))
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "clean: remove %s: %v\n", full, err)
			return 1
		}
		fmt.Fprintln(stdout, full)
	}
	for _, rel := range manifestFiles {
		full := filepath.Join(*out, filepath.FromSlash(rel))
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "clean: remove manifest %s: %v\n", full, err)
			return 1
		}
		fmt.Fprintln(stdout, full)
	}
	if err := manifest.RemoveEmptyParents(*out, toDelete); err != nil {
		fmt.Fprintf(stderr, "clean: %v\n", err)
		return 1
	}
	if err := manifest.RemoveEmptyDirs(*out, manifest.ManifestsDir(*target)); err != nil {
		fmt.Fprintf(stderr, "clean: %v\n", err)
		return 1
	}
	if err := manifest.RemoveEmptyDirs(*out, filepath.Join(manifest.TargetRoot(*target), ".agentflow")); err != nil {
		fmt.Fprintf(stderr, "clean: %v\n", err)
		return 1
	}
	if err := manifest.RemoveEmptyDirs(*out, manifest.TargetRoot(*target)); err != nil {
		fmt.Fprintf(stderr, "clean: %v\n", err)
		return 1
	}

	return 0
}

func shouldSkipClean(art manifest.Artifact, m *manifest.Manifest, owners map[string][]string, report manifest.DriftReport, force bool, stderr io.Writer) bool {
	for _, mod := range report.Modified {
		if mod.Path == art.Path && !force {
			fmt.Fprintf(stderr, "warning AF310: skip modified artifact %s (use --force to delete)\n", art.Path)
			return true
		}
	}
	for _, unreadable := range report.Unreadable {
		if unreadable.Path == art.Path && !force {
			fmt.Fprintf(stderr, "warning AF314: skip unreadable artifact %s (use --force to delete)\n", art.Path)
			return true
		}
	}
	if other, ok := manifest.OtherOwner(art.Path, m.Source.Path, owners); ok && !force {
		fmt.Fprintf(stderr, "warning AF311: skip %s still owned by %s (use --force to delete)\n", art.Path, other)
		return true
	}
	return false
}

func uniqueSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
