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
	printJSON := flag.Bool("print-json", false, "print the generated JSON manifest to stdout")
	printMarkdown := flag.Bool("print-md", false, "print the generated Markdown report to stdout")
	printHeapTagsJSON := flag.Bool("print-heap-tags-json", false, "print the generated heap tag gap JSON report to stdout")
	printHeapTagsMarkdown := flag.Bool("print-heap-tags-md", false, "print the generated heap tag gap Markdown report to stdout")
	flag.Parse()

	manifest, err := coverage.BuildManifest()
	if err != nil {
		fail(err)
	}
	if *printJSON {
		fmt.Print(coverage.RenderManifestJSON(manifest))
		return
	}
	if *printMarkdown {
		fmt.Print(coverage.RenderMarkdown(manifest))
		return
	}
	heapTags, err := coverage.BuildHeapTagReport()
	if err != nil {
		fail(err)
	}
	if *printHeapTagsJSON {
		fmt.Print(coverage.RenderHeapTagReportJSON(heapTags))
		return
	}
	if *printHeapTagsMarkdown {
		fmt.Print(coverage.RenderHeapTagReportMarkdown(heapTags))
		return
	}

	files := coverage.ArtifactContents(manifest)
	for path, content := range coverage.HeapTagArtifactContents(heapTags) {
		files[path] = content
	}
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
