// Package golden is a tiny helper for golden-file tests.
//
// Usage:
//
//	golden.Assert(t, "testdata/golden/foo.json", got)
//
// If the file at path does not exist, or if the UPDATE_GOLDEN environment
// variable is set to a non-empty value, the golden file is (re)written with
// got and the test passes. Otherwise, got is compared to the file contents
// byte-for-byte; on mismatch, the test fails with a unified-diff hint.
//
// Intended for use from _test.go files only. Keep golden content small,
// textual, and deterministic.
package golden

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Assert compares got against the golden file at path. Paths are interpreted
// relative to the calling test's package directory, matching the standard
// Go testdata convention.
//
// The UPDATE_GOLDEN env var (any non-empty value) triggers regeneration
// instead of comparison; useful for deliberate updates via
// `UPDATE_GOLDEN=1 go test ./...`.
func Assert(t *testing.T, path string, got []byte) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") != "" {
		writeGolden(t, path, got)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Logf("golden file %q missing; writing initial version (set UPDATE_GOLDEN to regenerate in future)", path)
			writeGolden(t, path, got)
			return
		}
		t.Fatalf("read golden %q: %v", path, err)
	}

	if bytes.Equal(got, want) {
		return
	}

	t.Fatalf("golden mismatch for %q (len got=%d want=%d)\n  hint: re-run with UPDATE_GOLDEN=1 to accept changes\n--- got (first 400 bytes) ---\n%s\n--- want (first 400 bytes) ---\n%s",
		path, len(got), len(want), preview(got, 400), preview(want, 400))
}

func writeGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir golden dir: %v", err)
	}
	if err := os.WriteFile(path, got, 0o644); err != nil {
		t.Fatalf("write golden %q: %v", path, err)
	}
	t.Logf("wrote golden %q (%d bytes)", path, len(got))
}

func preview(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
