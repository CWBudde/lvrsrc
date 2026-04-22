package golden_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/golden"
)

func TestAssertCreatesMissingGolden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.golden")

	golden.Assert(t, path, []byte("hello\n"))

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file was not created: %v", err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("golden content = %q, want %q", got, "hello\n")
	}
}

func TestAssertPassesOnMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "match.golden")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatalf("seed golden: %v", err)
	}

	golden.Assert(t, path, []byte("same")) // should not fail
}

func TestAssertUpdateEnvOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.golden")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed golden: %v", err)
	}

	t.Setenv("UPDATE_GOLDEN", "1")
	golden.Assert(t, path, []byte("new"))

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("golden content = %q, want %q", got, "new")
	}
}
