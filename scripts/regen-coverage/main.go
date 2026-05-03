// regen-coverage rebuilds the docs/generated/* coverage and
// heap-tag-gaps artifacts from the current testdata/corpus state.
//
// Usage:
//
//	go run ./scripts/regen-coverage
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/CWBudde/lvrsrc/internal/coverage"
)

func main() {
	root := repoRoot()
	if err := os.Chdir(root); err != nil {
		fmt.Fprintf(os.Stderr, "chdir %s: %v\n", root, err)
		os.Exit(1)
	}

	manifest, err := coverage.BuildManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "BuildManifest: %v\n", err)
		os.Exit(1)
	}
	for path, content := range coverage.ArtifactContents(manifest) {
		writeArtifact(path, content)
	}

	tagReport, err := coverage.BuildHeapTagReport()
	if err != nil {
		fmt.Fprintf(os.Stderr, "BuildHeapTagReport: %v\n", err)
		os.Exit(1)
	}
	for path, content := range coverage.HeapTagArtifactContents(tagReport) {
		writeArtifact(path, content)
	}
}

func writeArtifact(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", path, len(content))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "cannot determine caller location")
		os.Exit(1)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
