package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestBoundsWidthHeight(t *testing.T) {
	b := Bounds{Left: 10, Top: 20, Right: 130, Bottom: 80}
	if got, want := b.Width(), int16(120); got != want {
		t.Errorf("Width() = %d, want %d", got, want)
	}
	if got, want := b.Height(), int16(60); got != want {
		t.Errorf("Height() = %d, want %d", got, want)
	}
}

func TestHeapBoundsHappyPath(t *testing.T) {
	// OF__bounds leaf with left=10, top=20, right=300, bottom=-1
	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:    int32(heap.FieldTagBounds),
				Scope:  "leaf",
				Parent: -1,
				Content: []byte{
					0x00, 0x0A, 0x00, 0x14, 0x01, 0x2C, 0xFF, 0xFF,
				},
			},
		},
		Roots: []int{0},
	}
	got, ok := HeapBounds(tree, 0)
	if !ok {
		t.Fatal("HeapBounds() ok = false, want true")
	}
	want := Bounds{Left: 10, Top: 20, Right: 300, Bottom: -1}
	if got != want {
		t.Errorf("HeapBounds() = %+v, want %+v", got, want)
	}
}

func TestHeapBoundsRejectsWrongTag(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:     int32(heap.FieldTagBgColor),
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0x00, 0x0A, 0x00, 0x14, 0x01, 0x2C, 0xFF, 0xFF},
			},
		},
		Roots: []int{0},
	}
	if _, ok := HeapBounds(tree, 0); ok {
		t.Error("HeapBounds() on non-bounds tag returned ok=true, want false")
	}
}

func TestHeapBoundsRejectsBadContentLength(t *testing.T) {
	for _, length := range []int{0, 4, 7, 9, 16} {
		tree := HeapTree{
			Nodes: []HeapNode{
				{
					Tag:     int32(heap.FieldTagBounds),
					Scope:   "leaf",
					Parent:  -1,
					Content: make([]byte, length),
				},
			},
			Roots: []int{0},
		}
		if _, ok := HeapBounds(tree, 0); ok {
			t.Errorf("HeapBounds() on %d-byte content returned ok=true, want false", length)
		}
	}
}

func TestHeapBoundsRejectsOutOfRangeIndex(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{}, Roots: []int{}}
	for _, idx := range []int{-1, 0, 1, 100} {
		if _, ok := HeapBounds(tree, idx); ok {
			t.Errorf("HeapBounds(idx=%d) on empty tree returned ok=true, want false", idx)
		}
	}
}

func TestHeapRectAcceptsKnownRectangleTags(t *testing.T) {
	rectTags := []heap.FieldTag{
		heap.FieldTagBounds,
		heap.FieldTagCallerGlyphBounds,
		heap.FieldTagContRect,
		heap.FieldTagDBounds,
		heap.FieldTagDocBounds,
		heap.FieldTagDynBounds,
		heap.FieldTagGrowAreaBounds,
		heap.FieldTagHoodBounds,
		heap.FieldTagIconBounds,
		heap.FieldTagPBounds,
		heap.FieldTagSizeRect,
		heap.FieldTagSubVIGlyphBounds,
		heap.FieldTagTermBounds,
		heap.FieldTagTotalBounds,
		heap.FieldTagStateBounds,
		heap.FieldTagIntensityGraphBounds,
	}
	want := Bounds{Left: -4, Top: 7, Right: 300, Bottom: 512}
	content := []byte{0xFF, 0xFC, 0x00, 0x07, 0x01, 0x2C, 0x02, 0x00}

	for _, tag := range rectTags {
		tree := HeapTree{
			Nodes: []HeapNode{{
				Tag:     int32(tag),
				Scope:   "leaf",
				Parent:  -1,
				Content: content,
			}},
			Roots: []int{0},
		}
		if !IsHeapRectTag(int32(tag)) {
			t.Fatalf("IsHeapRectTag(%s) = false, want true", tag)
		}
		got, ok := HeapRect(tree, 0)
		if !ok {
			t.Fatalf("HeapRect(%s) ok = false, want true", tag)
		}
		if got != want {
			t.Errorf("HeapRect(%s) = %+v, want %+v", tag, got, want)
		}
		got, ok = HeapRectForTag(tree, 0, int32(tag))
		if !ok {
			t.Fatalf("HeapRectForTag(%s) ok = false, want true", tag)
		}
		if got != want {
			t.Errorf("HeapRectForTag(%s) = %+v, want %+v", tag, got, want)
		}
	}
}

func TestHeapRectRejectsUnknownTagAndBadLength(t *testing.T) {
	unknownTag := int32(heap.FieldTagBgColor)
	if IsHeapRectTag(unknownTag) {
		t.Fatalf("IsHeapRectTag(%s) = true, want false", heap.FieldTag(unknownTag))
	}

	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:     unknownTag,
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0, 1, 0, 2, 0, 3, 0, 4},
			},
			{
				Tag:     int32(heap.FieldTagContRect),
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0, 1, 0, 2},
			},
		},
		Roots: []int{0, 1},
	}
	if _, ok := HeapRect(tree, 0); ok {
		t.Error("HeapRect() on unknown tag returned ok=true, want false")
	}
	if _, ok := HeapRectForTag(tree, 0, unknownTag); ok {
		t.Error("HeapRectForTag() on unknown tag returned ok=true, want false")
	}
	if _, ok := HeapRect(tree, 1); ok {
		t.Error("HeapRect() on bad rectangle length returned ok=true, want false")
	}
	if _, ok := HeapRectForTag(tree, 1, int32(heap.FieldTagContRect)); ok {
		t.Error("HeapRectForTag() on bad rectangle length returned ok=true, want false")
	}
}

func TestFindRectChildLocatesRequestedTag(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2, 3}},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: []byte{0, 1, 0, 2, 0, 3, 0, 4},
			},
			{
				Tag:     int32(heap.FieldTagDBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: []byte{0, 5, 0, 7, 0, 0x6E, 0, 0x4D},
			},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}
	got, ok := FindRectChild(tree, 0, int32(heap.FieldTagDBounds))
	if !ok {
		t.Fatal("FindRectChild() ok = false, want true")
	}
	want := Bounds{Left: 5, Top: 7, Right: 110, Bottom: 77}
	if got != want {
		t.Errorf("FindRectChild() = %+v, want %+v", got, want)
	}
	if _, ok := FindRectChild(tree, 0, int32(heap.FieldTagBgColor)); ok {
		t.Error("FindRectChild() on non-rectangle tag returned ok=true, want false")
	}
}

func TestFindBoundsChildLocatesOFBoundsLeafAmongSiblings(t *testing.T) {
	// Synthetic control node (parent at index 0) with three children:
	//   1: OF__bgColor leaf  (not bounds)
	//   2: OF__bounds  leaf  ← target
	//   3: OF__objFlags leaf (not bounds)
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2, 3}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
			{
				Tag:    int32(heap.FieldTagBounds),
				Scope:  "leaf",
				Parent: 0,
				Content: []byte{
					0x00, 0x05, 0x00, 0x07, 0x00, 0x6E, 0x00, 0x4D,
				},
			},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}
	got, ok := FindBoundsChild(tree, 0)
	if !ok {
		t.Fatal("FindBoundsChild() ok = false, want true")
	}
	want := Bounds{Left: 5, Top: 7, Right: 110, Bottom: 77}
	if got != want {
		t.Errorf("FindBoundsChild() = %+v, want %+v", got, want)
	}
}

func TestFindBoundsChildReturnsFalseWhenNoBoundsSibling(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}
	if _, ok := FindBoundsChild(tree, 0); ok {
		t.Error("FindBoundsChild() with no OF__bounds sibling returned ok=true, want false")
	}
}

// TestHeapBoundsCorpusCoverage exercises HeapBounds against every
// OF__bounds leaf in every FPHb/BDHb tree of the corpus. It records
// per-fixture and aggregate decoding rates, and asserts non-empty
// coverage so a regression in tag classification or content sizing is
// caught here. It is informational rather than strict — coordinates are
// only sanity-checked for non-pathological extremes (≤ ±10000).
func TestHeapBoundsCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalBoundsLeaves := 0
	totalDecoded := 0
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
			continue
		}
		m, _ := DecodeKnownResources(f)
		for _, getter := range []func() (HeapTree, bool){m.FrontPanel, m.BlockDiagram} {
			tree, ok := getter()
			if !ok {
				continue
			}
			exercised++
			for i, n := range tree.Nodes {
				if n.Tag != int32(heap.FieldTagBounds) || n.Scope != "leaf" {
					continue
				}
				totalBoundsLeaves++
				b, ok := HeapBounds(tree, i)
				if !ok {
					continue
				}
				totalDecoded++
				for _, v := range []int16{b.Left, b.Top, b.Right, b.Bottom} {
					if v < -10000 || v > 10000 {
						t.Errorf("%s node %d: extreme bounds coord %d in %+v",
							e.Name(), i, v, b)
					}
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	if totalBoundsLeaves == 0 {
		t.Fatalf("found 0 OF__bounds leaves across %d trees — tag classification regressed", exercised)
	}
	if totalDecoded != totalBoundsLeaves {
		t.Fatalf("HeapBounds decoded %d/%d OF__bounds leaves — bounds payload format inconsistent",
			totalDecoded, totalBoundsLeaves)
	}
	t.Logf("HeapBounds: %d/%d OF__bounds leaves decoded across %d trees",
		totalDecoded, totalBoundsLeaves, exercised)
}

// TestHeapRectCorpusCoverage exercises the generic rectangle accessor
// against every registered rectangle-like field tag in the corpus.
func TestHeapRectCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalRectLeaves := 0
	totalDecoded := 0
	exercised := 0
	countsByTag := map[int32]int{}
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
			continue
		}
		m, _ := DecodeKnownResources(f)
		for _, getter := range []func() (HeapTree, bool){m.FrontPanel, m.BlockDiagram} {
			tree, ok := getter()
			if !ok {
				continue
			}
			exercised++
			for i, n := range tree.Nodes {
				if n.Scope != "leaf" || !IsHeapRectTag(n.Tag) {
					continue
				}
				totalRectLeaves++
				countsByTag[n.Tag]++
				b, ok := HeapRect(tree, i)
				if !ok {
					t.Fatalf("%s node %d tag %s: generic rectangle decode failed for %d-byte payload",
						e.Name(), i, heap.FieldTag(n.Tag), len(n.Content))
				}
				totalDecoded++
				for _, v := range []int16{b.Left, b.Top, b.Right, b.Bottom} {
					if v < -10000 || v > 10000 {
						t.Errorf("%s node %d tag %s: extreme rect coord %d in %+v",
							e.Name(), i, heap.FieldTag(n.Tag), v, b)
					}
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	if totalRectLeaves == 0 {
		t.Fatalf("found 0 rectangle-like leaves across %d trees", exercised)
	}
	if totalDecoded != totalRectLeaves {
		t.Fatalf("HeapRect decoded %d/%d rectangle-like leaves", totalDecoded, totalRectLeaves)
	}
	t.Logf("HeapRect: %d/%d rectangle-like leaves decoded across %d trees; tags=%v",
		totalDecoded, totalRectLeaves, exercised, countsByTag)
}
