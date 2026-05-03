package render

import (
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// TestSceneCoversAllHeapWidgetTerminals (Phase 16.4 A1) asserts every
// heap-side WidgetKindTerminal node has a matching NodeKindTerminal
// scene entry — the regression that was missing pre-A1 caused wires
// to skip with MissingFromScene. After A2 the scene may carry
// additional NodeKindTerminal entries for per-endpoint canonical
// anchors, so we test set membership rather than count equality.
func TestSceneCoversAllHeapWidgetTerminals(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "testdata", "corpus")
	matches, err := filepath.Glob(filepath.Join(corpusDir, "*.vi"))
	if err != nil {
		t.Fatalf("glob corpus: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no corpus VIs found at %s", corpusDir)
	}
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			file, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			model, _ := lvvi.DecodeKnownResources(file)
			tree, ok := model.BlockDiagram()
			if !ok {
				t.Skip("no BD heap")
			}
			scene := ProjectHeapTree(tree, ViewBlockDiagram)
			sceneTerminalHeapIdx := map[int]bool{}
			for _, n := range scene.Nodes {
				if n.Kind == NodeKindTerminal {
					sceneTerminalHeapIdx[n.HeapIndex] = true
				}
			}
			for i, n := range tree.Nodes {
				if n.Scope == "open" && lvvi.WidgetKindForNode(n) == lvvi.WidgetKindTerminal {
					if !sceneTerminalHeapIdx[i] {
						t.Errorf("heap[%d] %s is WidgetKindTerminal but not projected as NodeKindTerminal in the scene",
							i, lvvi.HeapTagName(n))
					}
				}
			}
		})
	}
}

// TestNestedTerminalsHaveUniqueHeapIndex guards against the regression
// where two scene terminals collide on the same HeapIndex (which would
// reintroduce the wire-collapse bug because terminalByHeap would lose
// entries).
func TestNestedTerminalsHaveUniqueHeapIndex(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "testdata", "corpus")
	matches, err := filepath.Glob(filepath.Join(corpusDir, "*.vi"))
	if err != nil {
		t.Fatalf("glob corpus: %v", err)
	}
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			file, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			model, _ := lvvi.DecodeKnownResources(file)
			tree, ok := model.BlockDiagram()
			if !ok {
				t.Skip("no BD heap")
			}
			scene := ProjectHeapTree(tree, ViewBlockDiagram)
			seen := map[int]int{}
			for i, n := range scene.Nodes {
				if n.Kind != NodeKindTerminal {
					continue
				}
				if prev, dup := seen[n.HeapIndex]; dup {
					t.Errorf("scene[%d] and scene[%d] both reference heap[%d] — wire resolver will lose the second mapping",
						prev, i, n.HeapIndex)
				}
				seen[n.HeapIndex] = i
			}
		})
	}
}
