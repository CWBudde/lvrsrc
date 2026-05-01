package corpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type matrixFile struct {
	SchemaVersion int             `json:"schemaVersion"`
	Fixtures      []matrixFixture `json:"fixtures"`
	Gaps          []matrixGap     `json:"gaps"`
	OracleTargets []string        `json:"oracleTargets"`
}

type matrixFixture struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Family     string   `json:"family"`
	Focus      []string `json:"focus"`
	Hypothesis string   `json:"hypothesis"`
}

type matrixGap struct {
	Feature        string   `json:"feature"`
	Priority       string   `json:"priority"`
	NeededFixtures []string `json:"neededFixtures"`
	Notes          string   `json:"notes"`
}

type deltasFile struct {
	SchemaVersion int            `json:"schemaVersion"`
	Deltas        []reverseDelta `json:"deltas"`
}

type reverseDelta struct {
	ID         string   `json:"id"`
	Left       string   `json:"left"`
	Right      string   `json:"right"`
	Topic      string   `json:"topic"`
	Focus      []string `json:"focus"`
	Hypothesis string   `json:"hypothesis"`
	Evidence   string   `json:"evidence"`
	Status     string   `json:"status"`
}

func TestCorpusMatrixCoversFixturesAndGaps(t *testing.T) {
	m := readMatrix(t)
	if m.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", m.SchemaVersion)
	}
	if len(m.OracleTargets) == 0 {
		t.Fatal("OracleTargets is empty")
	}

	entries, err := os.ReadDir(Dir())
	if err != nil {
		t.Fatalf("ReadDir(corpus): %v", err)
	}
	wantFixtures := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".vi" && ext != ".ctl" {
			continue
		}
		wantFixtures[entry.Name()] = strings.TrimPrefix(ext, ".")
	}

	seen := map[string]struct{}{}
	for _, f := range m.Fixtures {
		if f.Name == "" || f.Kind == "" || f.Family == "" || len(f.Focus) == 0 || f.Hypothesis == "" {
			t.Fatalf("incomplete matrix fixture entry: %+v", f)
		}
		if _, dup := seen[f.Name]; dup {
			t.Fatalf("duplicate matrix fixture entry %q", f.Name)
		}
		seen[f.Name] = struct{}{}
		wantKind, ok := wantFixtures[f.Name]
		if !ok {
			t.Fatalf("matrix fixture %q does not exist in testdata/corpus", f.Name)
		}
		if f.Kind != wantKind {
			t.Fatalf("matrix fixture %q kind = %q, want %q", f.Name, f.Kind, wantKind)
		}
	}

	for name := range wantFixtures {
		if _, ok := seen[name]; !ok {
			t.Fatalf("corpus fixture %q is missing from matrix.json", name)
		}
	}

	requiredGaps := []string{
		"labels",
		"captions",
		"fonts",
		"colors",
		"scales",
		"decorations",
		"arrays",
		"graphs",
		"structures",
		"refnums",
		"variants",
		"subVI calls",
		"event structures",
		"disabled diagrams",
		"manual wires",
		"multi-elbow auto wires",
		"version span",
	}
	gaps := map[string]matrixGap{}
	for _, gap := range m.Gaps {
		if gap.Feature == "" || gap.Priority == "" || len(gap.NeededFixtures) == 0 || gap.Notes == "" {
			t.Fatalf("incomplete matrix gap entry: %+v", gap)
		}
		gaps[gap.Feature] = gap
	}
	for _, feature := range requiredGaps {
		if _, ok := gaps[feature]; !ok {
			t.Fatalf("matrix gap %q is missing", feature)
		}
	}
}

func TestReverseDeltasReferenceCorpusFixtures(t *testing.T) {
	deltas := readDeltas(t)
	if deltas.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", deltas.SchemaVersion)
	}
	if len(deltas.Deltas) < 10 {
		t.Fatalf("len(Deltas) = %d, want at least 10", len(deltas.Deltas))
	}

	entries, err := os.ReadDir(Dir())
	if err != nil {
		t.Fatalf("ReadDir(corpus): %v", err)
	}
	fixtures := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".vi" || ext == ".ctl" {
			fixtures[entry.Name()] = struct{}{}
		}
	}

	seenIDs := map[string]struct{}{}
	topics := map[string]int{}
	for _, delta := range deltas.Deltas {
		if delta.ID == "" || delta.Left == "" || delta.Right == "" || delta.Topic == "" || len(delta.Focus) == 0 || delta.Hypothesis == "" || delta.Evidence == "" || delta.Status == "" {
			t.Fatalf("incomplete reverse delta entry: %+v", delta)
		}
		if _, dup := seenIDs[delta.ID]; dup {
			t.Fatalf("duplicate reverse delta id %q", delta.ID)
		}
		seenIDs[delta.ID] = struct{}{}
		if _, ok := fixtures[delta.Left]; !ok {
			t.Fatalf("reverse delta %q left fixture %q does not exist", delta.ID, delta.Left)
		}
		if _, ok := fixtures[delta.Right]; !ok {
			t.Fatalf("reverse delta %q right fixture %q does not exist", delta.ID, delta.Right)
		}
		topics[delta.Topic]++
	}

	for _, topic := range []string{"wire-topology", "type-system", "widget-and-terminal-kind"} {
		if topics[topic] == 0 {
			t.Fatalf("reverse deltas have no entries for topic %q", topic)
		}
	}
}

func readMatrix(t *testing.T) matrixFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(), "docs", "corpus-matrix.json"))
	if err != nil {
		t.Fatalf("ReadFile(docs/corpus-matrix.json): %v", err)
	}
	var m matrixFile
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal(docs/corpus-matrix.json): %v", err)
	}
	return m
}

func readDeltas(t *testing.T) deltasFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(), "docs", "reverse-deltas.json"))
	if err != nil {
		t.Fatalf("ReadFile(docs/reverse-deltas.json): %v", err)
	}
	var d deltasFile
	if err := json.Unmarshal(data, &d); err != nil {
		t.Fatalf("Unmarshal(docs/reverse-deltas.json): %v", err)
	}
	return d
}
