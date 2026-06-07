package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestHeapTagNameAtContextAware verifies the context-dependent naming
// fix across the whole corpus:
//
//   - tag-0 leaves are resolved by their enclosing object class, not
//     blindly named SL__fontRun (the old context-free bug);
//   - generic field tags (objFlags=172, howGrow=106, partID=192) always
//     resolve to their OF__ name, never to the colliding SL_CLASS_TAGS
//     name (SL__grouper, SL__extFunc, SL__exprNode).
func TestHeapTagNameAtContextAware(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}

	// Generic field tags and the wrong (class) name the old resolver gave.
	genericFields := map[int32]struct{ field, wrongClass string }{
		172: {"OF__objFlags", "SL__grouper"},
		106: {"OF__howGrow", "SL__extFunc"},
		192: {"OF__partID", "SL__exprNode"},
	}

	var (
		treesSeen      int
		tag0Image      int
		tag0FontRunMis int
		genericChecked int
	)
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
			treesSeen++
			for i, n := range tree.Nodes {
				name := HeapTagNameAt(tree, i)

				// tag-0 leaves must never be SL__fontRun unless the
				// enclosing object really is an SL__fontRun (none in corpus).
				if n.Tag == 0 && n.Scope == "leaf" {
					parentClass := ParentTopClass(tree, n.Parent)
					if name == "SL__fontRun" && parentClass != heap.ClassTagFontRun {
						tag0FontRunMis++
					}
					if parentClass == heap.ClassTagImage {
						tag0Image++
						if name != "OF__ImageResID" {
							t.Errorf("%s: tag-0 leaf under SL__Image = %q, want OF__ImageResID", e.Name(), name)
						}
					}
				}

				// Generic field leaves must resolve to their OF__ name.
				if want, ok := genericFields[n.Tag]; ok && n.Scope == "leaf" {
					if _, isClass := HeapNodeClass(n); !isClass {
						genericChecked++
						if name != want.field {
							t.Errorf("%s: tag %d leaf = %q, want %q (must not be %q)",
								e.Name(), n.Tag, name, want.field, want.wrongClass)
						}
					}
				}
			}
		}
	}

	if treesSeen == 0 {
		t.Skip("no heap trees in corpus")
	}
	if tag0FontRunMis != 0 {
		t.Errorf("found %d tag-0 leaves still mislabeled SL__fontRun", tag0FontRunMis)
	}
	if tag0Image == 0 {
		t.Error("expected some tag-0 leaves under SL__Image in the corpus, found none")
	}
	if genericChecked == 0 {
		t.Error("expected some generic field leaves (objFlags/howGrow/partID), found none")
	}
	t.Logf("context-aware naming validated across %d heap trees: %d tag-0/Image leaves, %d generic field leaves",
		treesSeen, tag0Image, genericChecked)
}
