package heap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestDecodeEntryLeafNoAttrsNoContent(t *testing.T) {
	// cmd = sizeSpec=0, hasAttrList=0, scope=TagLeaf=1, rawTagID=42 (= 31+11)
	// cmd[0] high byte: 0b 000_0_01_00 = 0x04, low bits of rawTagID 42 are 0
	// So byte 0 = (0<<5) | (0<<4) | (1<<2) | (rawTagID>>8 & 3) = 0x04
	// Byte 1 = rawTagID & 0xFF = 42
	payload := []byte{0x04, 42}
	node, n, err := decodeEntry(payload)
	if err != nil {
		t.Fatalf("decodeEntry: %v", err)
	}
	if n != 2 {
		t.Errorf("consumed = %d, want 2", n)
	}
	if node.Scope != NodeScope(NodeScopeTagLeaf) {
		t.Errorf("Scope = %v, want TagLeaf", node.Scope)
	}
	if node.Tag != 42-31 {
		t.Errorf("Tag = %d, want %d", node.Tag, 42-31)
	}
	if node.SizeSpec != 0 {
		t.Errorf("SizeSpec = %d, want 0", node.SizeSpec)
	}
	if !node.IsBool() || node.BoolValue() != false {
		t.Errorf("expected bool false, got IsBool=%v Value=%v", node.IsBool(), node.BoolValue())
	}
	if node.HasExplicitTag {
		t.Error("HasExplicitTag = true, want false")
	}
}

func TestDecodeEntryBoolTrue(t *testing.T) {
	// sizeSpec=7 (bool true), scope=TagLeaf
	payload := []byte{0xE4, 50} // 0b 111_0_01_00 = 0xE4
	node, _, err := decodeEntry(payload)
	if err != nil {
		t.Fatalf("decodeEntry: %v", err)
	}
	if node.SizeSpec != 7 {
		t.Errorf("SizeSpec = %d, want 7", node.SizeSpec)
	}
	if !node.BoolValue() {
		t.Error("BoolValue = false, want true")
	}
}

func TestDecodeEntryFixedSizeContent(t *testing.T) {
	// sizeSpec=4 (4 bytes content), no attrs, scope=TagLeaf, tagID=80 (raw=111)
	// byte 0 = (4<<5) | (0<<4) | (1<<2) | 0 = 0x84
	// byte 1 = 111
	payload := []byte{0x84, 111, 0xDE, 0xAD, 0xBE, 0xEF}
	node, n, err := decodeEntry(payload)
	if err != nil {
		t.Fatalf("decodeEntry: %v", err)
	}
	if n != 6 {
		t.Errorf("consumed = %d, want 6", n)
	}
	if node.SizeSpec != 4 {
		t.Errorf("SizeSpec = %d, want 4", node.SizeSpec)
	}
	want := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if string(node.Content) != string(want) {
		t.Errorf("Content = %x, want %x", node.Content, want)
	}
}

func TestDecodeEntryVarLengthContent(t *testing.T) {
	// sizeSpec=6 (var length), scope=TagLeaf, rawTagID=200
	// byte 0 = (6<<5) | (0<<4) | (1<<2) | 0 = 0xC4
	// byte 1 = 200
	// Then U124 length = 5 (single byte), then 5 bytes
	payload := []byte{0xC4, 200, 5, 'h', 'e', 'l', 'l', 'o'}
	node, n, err := decodeEntry(payload)
	if err != nil {
		t.Fatalf("decodeEntry: %v", err)
	}
	if n != 8 {
		t.Errorf("consumed = %d, want 8", n)
	}
	if string(node.Content) != "hello" {
		t.Errorf("Content = %q, want %q", node.Content, "hello")
	}
}

func TestDecodeEntryWithAttrs(t *testing.T) {
	// sizeSpec=0, hasAttrList=1, scope=TagOpen=0, rawTagID=100
	// byte 0 = (0<<5) | (1<<4) | (0<<2) | 0 = 0x10
	// byte 1 = 100
	// then U124 attr count = 1 (single byte 0x01)
	// then attribute: S124 id=5, S24 value=42
	payload := []byte{0x10, 100, 0x01, 5, 0, 42}
	node, n, err := decodeEntry(payload)
	if err != nil {
		t.Fatalf("decodeEntry: %v", err)
	}
	if n != 6 {
		t.Errorf("consumed = %d, want 6", n)
	}
	if node.Scope != NodeScope(NodeScopeTagOpen) {
		t.Errorf("Scope = %v, want TagOpen", node.Scope)
	}
	if len(node.Attribs) != 1 {
		t.Fatalf("len(Attribs) = %d, want 1", len(node.Attribs))
	}
	if node.Attribs[0].ID != 5 || node.Attribs[0].Value != 42 {
		t.Errorf("Attribs[0] = %+v, want {5, 42}", node.Attribs[0])
	}
}

func TestDecodeEntryExplicitTagEscape(t *testing.T) {
	// rawTagID = 1023 → escape; trailing int32 BE is the tag.
	// byte 0 = (0<<5) | (0<<4) | (1<<2) | 0x03 (high 2 bits of 1023) = 0x07
	// byte 1 = 0xFF (low 8 bits of 1023)
	// then 4 bytes BE int32 = -100
	payload := []byte{0x07, 0xFF, 0xFF, 0xFF, 0xFF, 0x9C}
	node, n, err := decodeEntry(payload)
	if err != nil {
		t.Fatalf("decodeEntry: %v", err)
	}
	if n != 6 {
		t.Errorf("consumed = %d, want 6", n)
	}
	if !node.HasExplicitTag {
		t.Error("HasExplicitTag = false, want true")
	}
	if node.Tag != -100 {
		t.Errorf("Tag = %d, want -100", node.Tag)
	}
}

func TestDecodeEntryRejectsTruncatedHeader(t *testing.T) {
	for _, payload := range [][]byte{nil, {0x04}} {
		if _, _, err := decodeEntry(payload); err == nil {
			t.Errorf("decodeEntry(%x) returned nil error", payload)
		}
	}
}

func TestDecodeEntryRejectsReservedSizeSpec5(t *testing.T) {
	// sizeSpec=5 is reserved; reject loudly so we surface the case if
	// real corpus data ever contains it.
	payload := []byte{0xA4, 100} // sizeSpec=5
	if _, _, err := decodeEntry(payload); err == nil {
		t.Error("decodeEntry of sizeSpec=5 returned nil error")
	}
}

func TestWalkOpenLeafCloseFormsTree(t *testing.T) {
	// Build a 3-entry stream: TagOpen(rawID 50) + TagLeaf(rawID 60) + TagClose(rawID 50).
	// All sizeSpec=0 (bool false), hasAttrList=0.
	open := []byte{0x00, 50}     // sizeSpec=0, hasAttr=0, scope=TagOpen=0
	leaf := []byte{0x04, 60}     // scope=TagLeaf=1
	close := []byte{0x08, 50}    // scope=TagClose=2
	payload := append(append(open, leaf...), close...)

	res, err := Walk(payload)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(res.Flat) != 3 {
		t.Fatalf("Flat = %d entries, want 3", len(res.Flat))
	}
	if len(res.Roots) != 2 {
		// Open + Close are top-level siblings; Leaf is under Open.
		t.Fatalf("Roots = %d, want 2 (Open, Close)", len(res.Roots))
	}
	if len(res.Roots[0].Children) != 1 {
		t.Fatalf("Open node children = %d, want 1", len(res.Roots[0].Children))
	}
	if res.Roots[0].Children[0].Scope != NodeScope(NodeScopeTagLeaf) {
		t.Errorf("Open node child scope = %v, want TagLeaf", res.Roots[0].Children[0].Scope)
	}
	if res.Roots[1].Scope != NodeScope(NodeScopeTagClose) {
		t.Errorf("Roots[1].Scope = %v, want TagClose", res.Roots[1].Scope)
	}
	if res.Roots[0].Children[0].Parent() == nil {
		t.Error("Leaf entry has nil parent, want Open node")
	}
}

func TestWalkRejectsTrailingBytes(t *testing.T) {
	// Trailing junk after a valid leaf entry must error.
	payload := []byte{0x04, 50, 0xAA}
	if _, err := Walk(payload); err == nil {
		t.Error("Walk with trailing bytes returned nil error")
	}
}

func TestWalkAcceptsEmpty(t *testing.T) {
	res, err := Walk(nil)
	if err != nil {
		t.Fatalf("Walk(nil): %v", err)
	}
	if len(res.Flat) != 0 || len(res.Roots) != 0 {
		t.Errorf("Walk(nil) = %d flat, %d roots; want both 0", len(res.Flat), len(res.Roots))
	}
}

func TestWalkCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalEntries := 0
	totalRoots := 0
	files := 0
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
		for _, block := range f.Blocks {
			if block.Type != "FPHb" && block.Type != "BDHb" {
				continue
			}
			for _, section := range block.Sections {
				env, err := DecodeEnvelope(section.Payload)
				if err != nil {
					t.Errorf("%s %s id=%d DecodeEnvelope: %v", e.Name(), block.Type, section.Index, err)
					continue
				}
				res, err := Walk(env.Content)
				if err != nil {
					t.Errorf("%s %s id=%d Walk: %v", e.Name(), block.Type, section.Index, err)
					continue
				}
				files++
				totalEntries += len(res.Flat)
				totalRoots += len(res.Roots)
			}
		}
	}
	if files == 0 {
		t.Skip("no heap sections in corpus")
	}
	t.Logf("walked %d heap streams: %d total entries, %d top-level roots", files, totalEntries, totalRoots)
}
