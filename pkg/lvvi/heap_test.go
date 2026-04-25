package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestModelFrontPanelReturnsFalseWhenNoFPHb(t *testing.T) {
	m, _ := DecodeKnownResources(&lvrsrc.File{})
	if _, ok := m.FrontPanel(); ok {
		t.Error("FrontPanel() ok = true on empty file, want false")
	}
}

func TestModelFrontPanelReturnsFalseOnNilReceiver(t *testing.T) {
	var m *Model
	if _, ok := m.FrontPanel(); ok {
		t.Error("FrontPanel() ok = true on nil receiver, want false")
	}
}

func TestModelFrontPanelOnCorpusHasConsistentTreeIndices(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalNodes := 0
	totalRoots := 0
	exercised := 0
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
		tree, ok := m.FrontPanel()
		if !ok {
			continue
		}
		exercised++
		totalNodes += len(tree.Nodes)
		totalRoots += len(tree.Roots)

		// Every Roots index points into Nodes and refers to a node with
		// Parent == -1.
		for _, ri := range tree.Roots {
			if ri < 0 || ri >= len(tree.Nodes) {
				t.Fatalf("%s: Roots index %d out of range [0,%d)", e.Name(), ri, len(tree.Nodes))
			}
			if got := tree.Nodes[ri].Parent; got != -1 {
				t.Fatalf("%s: Nodes[Roots[..]=%d].Parent = %d, want -1", e.Name(), ri, got)
			}
		}

		// Every node's Parent index is either -1 or a valid Nodes index;
		// every child index in Children is a valid Nodes index whose
		// Parent points back at the current node.
		for i, n := range tree.Nodes {
			if n.Parent == -1 {
				continue
			}
			if n.Parent < 0 || n.Parent >= len(tree.Nodes) {
				t.Fatalf("%s: Nodes[%d].Parent = %d, out of range", e.Name(), i, n.Parent)
			}
			parent := tree.Nodes[n.Parent]
			found := false
			for _, ci := range parent.Children {
				if ci == i {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s: Nodes[%d].Parent=%d but parent.Children does not contain %d",
					e.Name(), i, n.Parent, i)
			}
		}
		for i, n := range tree.Nodes {
			for _, ci := range n.Children {
				if ci < 0 || ci >= len(tree.Nodes) {
					t.Fatalf("%s: Nodes[%d].Children has out-of-range index %d", e.Name(), i, ci)
				}
				if tree.Nodes[ci].Parent != i {
					t.Fatalf("%s: Nodes[%d].Children[..]=%d but child.Parent=%d, want %d",
						e.Name(), i, ci, tree.Nodes[ci].Parent, i)
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no FPHb-bearing corpus VIs exercised")
	}
	t.Logf("FrontPanel: %d roots / %d total nodes across %d corpus VIs",
		totalRoots, totalNodes, exercised)
}

func TestModelBlockDiagramReturnsFalseWhenNoBDHb(t *testing.T) {
	m, _ := DecodeKnownResources(&lvrsrc.File{})
	if _, ok := m.BlockDiagram(); ok {
		t.Error("BlockDiagram() ok = true on empty file, want false")
	}
}

func TestModelBlockDiagramOnCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalNodes := 0
	exercised := 0
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
		tree, ok := m.BlockDiagram()
		if !ok {
			continue
		}
		exercised++
		totalNodes += len(tree.Nodes)
		// Same Parent/Children consistency invariants as FrontPanel.
		for i, n := range tree.Nodes {
			for _, ci := range n.Children {
				if tree.Nodes[ci].Parent != i {
					t.Fatalf("%s: BDHb child/parent mismatch at %d→%d", e.Name(), i, ci)
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no BDHb-bearing corpus VIs exercised")
	}
	t.Logf("BlockDiagram: %d total nodes across %d corpus VIs", totalNodes, exercised)
}

func TestHeapTagNameKnownAndFallback(t *testing.T) {
	// SystemTag: -3 → SL__object.
	if got := HeapTagName(HeapNode{Tag: -3, Scope: "open"}); got != "SL__object" {
		t.Errorf("HeapTagName(-3) = %q, want SL__object", got)
	}
	// Unknown positive tag falls back to a numeric label, not empty.
	got := HeapTagName(HeapNode{Tag: 99999, Scope: "leaf"})
	if got == "" {
		t.Error("HeapTagName(unknown) = empty, want a numeric fallback")
	}
}

func TestHeapTagNameResolvesCorpusOpenTags(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	for _, e := range entries {
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := DecodeKnownResources(f)
		tree, ok := m.FrontPanel()
		if !ok || len(tree.Nodes) == 0 {
			continue
		}
		// At least some open-scope nodes must resolve to a non-numeric
		// tag name; otherwise the resolver is useless.
		resolved := 0
		opens := 0
		for _, n := range tree.Nodes {
			if n.Scope != "open" {
				continue
			}
			opens++
			name := HeapTagName(n)
			// Numeric fallbacks contain "(" — e.g. "ClassTag(123)".
			if name != "" && !containsRune(name, '(') {
				resolved++
			}
		}
		if opens == 0 {
			continue
		}
		if resolved == 0 {
			t.Errorf("%s: HeapTagName resolved 0/%d open tags — resolver coverage is empty",
				e.Name(), opens)
		}
		t.Logf("%s: HeapTagName resolved %d/%d open-scope FPHb tags",
			e.Name(), resolved, opens)
		return
	}
	t.Skip("no FPHb-bearing corpus VI")
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func TestModelFrontPanelScopeStrings(t *testing.T) {
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
		tree, ok := m.FrontPanel()
		if !ok || len(tree.Nodes) == 0 {
			continue
		}
		seen := map[string]int{}
		for _, n := range tree.Nodes {
			seen[n.Scope]++
		}
		for s := range seen {
			switch s {
			case "open", "leaf", "close":
			default:
				t.Errorf("%s: unexpected Scope %q", e.Name(), s)
			}
		}
		// A real heap always has at least one open and matching close.
		if seen["open"] == 0 || seen["close"] == 0 {
			t.Errorf("%s: heap has no open/close nodes (scopes=%v)", e.Name(), seen)
		}
		return
	}
	t.Skip("no corpus VI with a decodable FPHb")
}
