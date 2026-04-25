package vctp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestFullTypeStrings(t *testing.T) {
	cases := []struct {
		t    FullType
		want string
	}{
		{FullTypeVoid, "Void"},
		{FullTypeNumInt32, "NumInt32"},
		{FullTypeBoolean, "Boolean"},
		{FullTypeString, "String"},
		{FullTypeArray, "Array"},
		{FullTypeCluster, "Cluster"},
		{FullTypeRefnum, "Refnum"},
		{FullTypeFunction, "Function"},
		{FullTypeTypeDef, "TypeDef"},
		{FullType(0xAB), "Type(0xab)"},
	}
	for _, tc := range cases {
		if got := tc.t.String(); got != tc.want {
			t.Errorf("FullType(%#x).String() = %q, want %q", uint8(tc.t), got, tc.want)
		}
	}
}

func TestParseInnerHandcrafted(t *testing.T) {
	// Two typedescs:
	//   1. Boolean (type 0x21), no flags, len=4 (header only)
	//   2. NumInt32 (type 0x03), HasLabel flag, len = 4 + label-stuff
	// Plus a top types list of length 1 pointing to flat ID 1.
	inflated := []byte{
		0, 0, 0, 2, // count = 2
		// TD 0: len=4, flags=0, type=0x21 (Boolean)
		0, 4, 0, 0x21,
		// TD 1: len=10, flags=0x40 (HasLabel), type=0x03 (NumInt32),
		// followed by a 5-byte label ("hello" with leading length 5; padded so total is 10)
		0, 10, 0x40, 0x03, 5, 'h', 'e', 'l', 'l', 'o',
		// Top types list: count = 1 (u2p2), then index = 1 (u2p2)
		0, 1,
		0, 1,
	}
	descs, tops, err := ParseInner(inflated)
	if err != nil {
		t.Fatalf("ParseInner: %v", err)
	}
	if len(descs) != 2 {
		t.Fatalf("len(descs) = %d, want 2", len(descs))
	}
	if descs[0].FullType != FullTypeBoolean {
		t.Errorf("descs[0].FullType = %v, want Boolean", descs[0].FullType)
	}
	if descs[1].FullType != FullTypeNumInt32 {
		t.Errorf("descs[1].FullType = %v, want NumInt32", descs[1].FullType)
	}
	if !descs[1].HasLabel {
		t.Errorf("descs[1].HasLabel = false, want true")
	}
	if descs[1].Label != "hello" {
		t.Errorf("descs[1].Label = %q, want %q", descs[1].Label, "hello")
	}
	if len(tops) != 1 || tops[0] != 1 {
		t.Errorf("tops = %v, want [1]", tops)
	}
}

func TestParseInnerRejectsTruncated(t *testing.T) {
	for _, payload := range [][]byte{{}, {0, 0, 0, 1}, {0, 0, 0, 1, 0, 4, 0}} {
		if _, _, err := ParseInner(payload); err == nil {
			t.Errorf("ParseInner(%d bytes) = nil error", len(payload))
		}
	}
}

func TestParseInnerEmpty(t *testing.T) {
	// Zero typedescs, zero top types.
	inflated := []byte{0, 0, 0, 0, 0, 0}
	descs, tops, err := ParseInner(inflated)
	if err != nil {
		t.Fatalf("ParseInner: %v", err)
	}
	if len(descs) != 0 || len(tops) != 0 {
		t.Errorf("got %d descs %d tops, want 0/0", len(descs), len(tops))
	}
}

func TestParseInnerCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalDescs := 0
	totalTops := 0
	files := 0
	typeHist := map[FullType]int{}
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
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				raw, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("%s VCTP Decode: %v", e.Name(), err)
				}
				v := raw.(Value)
				descs, tops, err := ParseInner(v.Inflated)
				if err != nil {
					t.Errorf("%s ParseInner: %v", e.Name(), err)
					continue
				}
				files++
				totalDescs += len(descs)
				totalTops += len(tops)
				for _, d := range descs {
					typeHist[d.FullType]++
				}
			}
		}
	}
	if files == 0 {
		t.Skip("no VCTP sections in corpus")
	}
	t.Logf("parsed %d VCTP sections totalling %d typedescs and %d top types; type histogram (top 5):", files, totalDescs, totalTops)
	// Print top-5 types in the histogram for sanity check
	type pair struct {
		t FullType
		n int
	}
	var pairs []pair
	for k, v := range typeHist {
		pairs = append(pairs, pair{k, v})
	}
	// crude top-5 by count
	for i := 0; i < 5 && i < len(pairs); i++ {
		max := i
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].n > pairs[max].n {
				max = j
			}
		}
		pairs[i], pairs[max] = pairs[max], pairs[i]
		t.Logf("  %s: %d", pairs[i].t, pairs[i].n)
	}
}
