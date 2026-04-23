package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/CWBudde/lvrsrc/internal/coverage"
)

func main() {
	check := flag.Bool("check", false, "verify checked-in coverage artifacts are current")
	flag.Parse()

	manifest, err := coverage.BuildManifest()
	if err != nil {
		fail(err)
	}

	files := coverage.ArtifactContents(manifest)
	if *check {
		if err := checkArtifacts(files); err != nil {
			fail(err)
		}
		return
	}
	if err := writeArtifacts(files); err != nil {
		fail(err)
	}
}

func writeArtifacts(files map[string]string) error {
	paths := sortedPaths(files)
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(files[path]), 0o644); err != nil {
			return fmt.Errorf("write %q: %w", path, err)
		}
	}
	return nil
}

func checkArtifacts(files map[string]string) error {
	paths := sortedPaths(files)
	for _, path := range paths {
		got, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}
		if string(got) != files[path] {
			return fmt.Errorf("%s is out of date; run go run ./internal/coverage/cmd/resourcecoverage", path)
		}
	}
	return nil
}

func sortedPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
