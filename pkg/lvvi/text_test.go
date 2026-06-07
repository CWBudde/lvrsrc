package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func openFixtureTree(t *testing.T, name string, bd bool) HeapTree {
	t.Helper()
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), name), lvrsrc.OpenOptions{})
	if err != nil {
		t.Skipf("fixture %s not available: %v", name, err)
	}
	m, _ := DecodeKnownResources(f)
	var tree HeapTree
	var ok bool
	if bd {
		tree, ok = m.BlockDiagram()
	} else {
		tree, ok = m.FrontPanel()
	}
	if !ok {
		t.Skipf("fixture %s has no %s heap", name, map[bool]string{true: "BD", false: "FP"}[bd])
	}
	return tree
}

// findLabelList returns the first label-list HeapText whose decoded lines
// equal want, or nil.
func findLabelList(texts []HeapText, want ...string) *HeapText {
	for i := range texts {
		ht := &texts[i]
		if ht.Kind != HeapTextLabelList || len(ht.Lines) != len(want) {
			continue
		}
		match := true
		for j := range want {
			if ht.Lines[j] != want[j] {
				match = false
				break
			}
		}
		if match {
			return ht
		}
	}
	return nil
}

func findString(texts []HeapText, want string) *HeapText {
	for i := range texts {
		if texts[i].Kind == HeapTextString && len(texts[i].Lines) == 1 && texts[i].Lines[0] == want {
			return &texts[i]
		}
	}
	return nil
}

// TestHeapTextsRingItemList checks that an enum/ring control's item text
// decodes as a label-list with the RING_TEXT role and a real owning class.
func TestHeapTextsRingItemList(t *testing.T) {
	tree := openFixtureTree(t, "reference-type.ctl", false)
	texts := HeapTexts(tree)
	ht := findLabelList(texts, "VI", "Project")
	if ht == nil {
		t.Fatalf("did not find ring item list [VI Project]; got %d texts", len(texts))
	}
	if ht.Role != PartIDRingText {
		t.Errorf("ring item list role = %s, want RING_TEXT", ht.Role)
	}
	if ht.OwnerName == "" {
		t.Errorf("ring item list has no owning class")
	}
}

// TestHeapTextsStateRing checks a multi-item enum list with several values.
func TestHeapTextsStateRing(t *testing.T) {
	tree := openFixtureTree(t, "mcp-creator-states.ctl", false)
	texts := HeapTexts(tree)
	ht := findLabelList(texts, "Wait For Event", "Update UI", "Update Server", "Load Project")
	if ht == nil {
		t.Fatalf("did not find state ring item list")
	}
	if ht.Role != PartIDRingText {
		t.Errorf("state ring role = %s, want RING_TEXT", ht.Role)
	}
}

// TestHeapTextsNameLabel checks that a control's name label decodes as a
// raw string with the NAME_LABEL role.
func TestHeapTextsNameLabel(t *testing.T) {
	tree := openFixtureTree(t, "Numeric255.vi", false)
	texts := HeapTexts(tree)
	ht := findString(texts, "Numeric")
	if ht == nil {
		t.Fatalf("did not find name label %q; got %d texts", "Numeric", len(texts))
	}
	if ht.TagName != "OF__text" {
		t.Errorf("name label tag = %s, want OF__text", ht.TagName)
	}
	if ht.Role != PartIDNameLabel {
		t.Errorf("name label role = %s, want NAME_LABEL", ht.Role)
	}
}

// TestHeapLabelListRoundTrip verifies the P-string decode is lossless: every
// SL__multiLabel buffer in the corpus re-encodes to its original bytes.
func TestHeapLabelListRoundTrip(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := DecodeKnownResources(f)
		for _, get := range []func() (HeapTree, bool){m.FrontPanel, m.BlockDiagram} {
			tree, ok := get()
			if !ok {
				continue
			}
			for i, n := range tree.Nodes {
				if n.Tag != int32(heap.FieldTagBuf) {
					continue
				}
				if ParentTopClass(tree, n.Parent) != heap.ClassTagMultiLabel {
					continue
				}
				lines, ok := HeapLabelListAt(tree, i)
				if !ok {
					t.Errorf("%s: multiLabel buf @%d failed to decode cleanly", e.Name(), i)
					continue
				}
				if got := encodePStrList(lines); string(got) != string(n.Content) {
					t.Errorf("%s: multiLabel buf @%d round-trip mismatch:\n got %v\nwant %v",
						e.Name(), i, got, n.Content)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Skip("no SL__multiLabel buffers in corpus")
	}
	t.Logf("round-tripped %d SL__multiLabel buffers", checked)
}

// TestHeapTextsCorpusInvariants sweeps the whole corpus asserting structural
// invariants of the text projection.
func TestHeapTextsCorpusInvariants(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	var strings, lists, nameLabels int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := DecodeKnownResources(f)
		for _, get := range []func() (HeapTree, bool){m.FrontPanel, m.BlockDiagram} {
			tree, ok := get()
			if !ok {
				continue
			}
			for _, ht := range HeapTexts(tree) {
				switch ht.Kind {
				case HeapTextString:
					strings++
					if len(ht.Lines) != 1 {
						t.Errorf("%s: string text @%d has %d lines, want 1", e.Name(), ht.NodeIndex, len(ht.Lines))
					}
					// OF__text is the content of whatever label/caption part
					// encloses it, so its role varies by context (NAME_LABEL,
					// BOOLEAN_TEXT, DIAGRAM_IDENTIFIER, or NO_PARTID for free
					// text). Just tally the name-label case here.
					if ht.TagName == "OF__text" && ht.Role == PartIDNameLabel {
						nameLabels++
					}
				case HeapTextLabelList:
					lists++
					if ht.Role != PartIDRingText && ht.Role != PartIDBooleanText {
						t.Errorf("%s: label-list @%d role = %s, want RING_TEXT or BOOLEAN_TEXT", e.Name(), ht.NodeIndex, ht.Role)
					}
				default:
					t.Errorf("%s: unexpected text kind %q", e.Name(), ht.Kind)
				}
			}
		}
	}
	if strings == 0 || lists == 0 {
		t.Errorf("expected both string (%d) and label-list (%d) texts in corpus", strings, lists)
	}
	if nameLabels == 0 {
		t.Error("expected some OF__text NAME_LABEL strings in corpus, found none")
	}
	t.Logf("corpus texts: %d strings (%d OF__text name labels), %d label-lists", strings, nameLabels, lists)
}

func encodePStrList(lines []string) []byte {
	var out []byte
	for _, s := range lines {
		out = append(out, byte(len(s)))
		out = append(out, s...)
	}
	return out
}
