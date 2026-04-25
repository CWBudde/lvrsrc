package oracle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// oracleDoc is the JSON payload emitted by scripts/gen-oracle.py.
type oracleDoc struct {
	Oracle     string             `json:"oracle"`
	SourcePath string             `json:"source_path"`
	FmtVersion int                `json:"fmt_version"`
	BlockCount int                `json:"block_count"`
	Blocks     []oracleBlockEntry `json:"blocks"`
}

type oracleBlockEntry struct {
	FourCC   string `json:"fourcc"`
	Sections int    `json:"sections"`
}

// repoRoot returns the absolute path to the lvrsrc repository root by
// walking up from this test file's location. Using runtime.Caller lets the
// test run regardless of the working directory chosen by `go test`.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/oracle/oracle_test.go → repo root is two levels up from internal/oracle.
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func loadOracle(t *testing.T, path string) oracleDoc {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read oracle %s: %v", path, err)
	}
	var doc oracleDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse oracle %s: %v", path, err)
	}
	return doc
}

// lvrsrcInventory returns (fourcc, section_count) pairs for f, sorted by
// fourcc to match the oracle's canonical ordering.
func lvrsrcInventory(f *lvrsrc.File) []oracleBlockEntry {
	out := make([]oracleBlockEntry, 0, len(f.Blocks))
	for _, b := range f.Blocks {
		out = append(out, oracleBlockEntry{FourCC: b.Type, Sections: len(b.Sections)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FourCC < out[j].FourCC })
	return out
}

// TestPylabviewOracleBlockSectionCounts walks testdata/oracle/**/*.json and,
// for each oracle, opens the matching corpus file with lvrsrc.Open and
// asserts that the (fourcc, section count) inventory lvrsrc reports matches
// what pylabview observed at baseline-generation time.
//
// Missing corpus files are skipped (corpus is user-supplied and may not be
// present in every checkout). Mismatches hard-fail so drift between lvrsrc
// and pylabview is caught before a release.
func TestPylabviewOracleBlockSectionCounts(t *testing.T) {
	root := repoRoot(t)
	oracleDir := filepath.Join(root, "testdata", "oracle")

	if _, err := os.Stat(oracleDir); errors.Is(err, fs.ErrNotExist) {
		t.Skipf("no oracle directory at %s; run scripts/gen-oracle.py to generate baselines", oracleDir)
	}

	var oracles []string
	err := filepath.WalkDir(oracleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".json") {
			oracles = append(oracles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk oracle dir: %v", err)
	}

	if len(oracles) == 0 {
		t.Skip("no oracle baselines found; run scripts/gen-oracle.py")
	}

	sort.Strings(oracles)

	for _, oraclePath := range oracles {
		rel, _ := filepath.Rel(root, oraclePath)
		t.Run(rel, func(t *testing.T) {
			doc := loadOracle(t, oraclePath)

			if doc.Oracle != "pylabview" {
				t.Fatalf("oracle %q: unknown oracle kind %q", rel, doc.Oracle)
			}

			corpusPath := filepath.Join(root, doc.SourcePath)
			if _, err := os.Stat(corpusPath); errors.Is(err, fs.ErrNotExist) {
				t.Skipf("corpus file missing: %s", doc.SourcePath)
			}

			f, err := lvrsrc.Open(corpusPath, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("lvrsrc.Open(%s): %v", doc.SourcePath, err)
			}

			got := lvrsrcInventory(f)
			want := append([]oracleBlockEntry(nil), doc.Blocks...)
			sort.Slice(want, func(i, j int) bool { return want[i].FourCC < want[j].FourCC })

			if len(got) != doc.BlockCount {
				t.Errorf("block_count mismatch: lvrsrc=%d oracle.block_count=%d", len(got), doc.BlockCount)
			}

			if !equalInventories(got, want) {
				t.Errorf(
					"block/section inventory mismatch for %s:\n  lvrsrc:    %s\n  pylabview: %s",
					doc.SourcePath,
					formatInventory(got),
					formatInventory(want),
				)
			}
		})
	}
}

func equalInventories(a, b []oracleBlockEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func formatInventory(entries []oracleBlockEntry) string {
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%s=%d", e.FourCC, e.Sections)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
