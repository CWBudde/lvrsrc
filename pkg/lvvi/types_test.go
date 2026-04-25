package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestModelTypesReturnsFalseWhenNoVCTP(t *testing.T) {
	m, _ := DecodeKnownResources(&lvrsrc.File{})
	if _, ok := m.Types(); ok {
		t.Error("Types() ok = true on empty file, want false")
	}
}

func TestModelTypesAndConnectorPaneOnCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalDescs := 0
	totalPanes := 0
	resolvedPanes := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("open %s: %v", e.Name(), err)
		}
		m, _ := DecodeKnownResources(f)
		descs, ok := m.Types()
		if !ok {
			continue
		}
		totalDescs += len(descs)

		pane, ok := m.ConnectorPane()
		if !ok {
			continue
		}
		totalPanes++
		if pane.HasPaneType {
			resolvedPanes++
		}
	}
	if totalDescs == 0 {
		t.Skip("no VCTP descs exercised")
	}
	t.Logf("Types: %d total typedescs across corpus; ConnectorPane: %d/%d resolved through VCTP",
		totalDescs, resolvedPanes, totalPanes)
}

func TestModelTypeAtIs1Based(t *testing.T) {
	// Find any corpus VI with at least one typedesc.
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := DecodeKnownResources(f)
		descs, ok := m.Types()
		if !ok || len(descs) == 0 {
			continue
		}

		// flatID 0 is reserved as "no type"; should return ok=false.
		if _, ok := m.TypeAt(0); ok {
			t.Errorf("TypeAt(0) ok = true, want false (0 is reserved)")
		}

		// flatID 1 should match descs[0].
		got, ok := m.TypeAt(1)
		if !ok {
			t.Fatalf("TypeAt(1) ok = false")
		}
		if got.Index != 0 {
			t.Errorf("TypeAt(1).Index = %d, want 0", got.Index)
		}

		// Out-of-range flatID returns ok=false.
		if _, ok := m.TypeAt(uint32(len(descs) + 1)); ok {
			t.Errorf("TypeAt(out of range) ok = true, want false")
		}
		return
	}
	t.Skip("no corpus VI with typedescs")
}
