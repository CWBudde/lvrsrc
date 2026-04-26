package libd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/libd"
	"github.com/CWBudde/lvrsrc/internal/codecs/linkobj"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestEntryTarget_Corpus mirrors the LIfp target sweep but for LIbd.
// The corpus surfaces both TDCC (typed) and IUVI (opaque), so both
// branches of the dispatcher run.
func TestEntryTarget_Corpus(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "testdata", "corpus")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}

	var sections, totalEntries, typedTargets int
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		ctx := codecs.Context{FileVersion: f.Header.FormatVersion, Kind: f.Kind}
		for _, blk := range f.Blocks {
			if blk.Type != "LIbd" {
				continue
			}
			for _, sec := range blk.Sections {
				out, err := (libd.Codec{}).Decode(ctx, sec.Payload)
				if err != nil {
					t.Errorf("%s: LIbd decode: %v", ent.Name(), err)
					continue
				}
				v := out.(libd.Value)
				sections++
				for i, e := range v.Entries {
					totalEntries++
					target, err := e.Target()
					if err != nil {
						t.Errorf("%s entry %d (%s): Target: %v", ent.Name(), i, e.LinkType, err)
						continue
					}
					body, secondary, err := linkobj.Encode(target)
					if err != nil {
						t.Errorf("%s entry %d (%s): re-encode: %v", ent.Name(), i, e.LinkType, err)
						continue
					}
					if !bytesEqual(body, e.Tail) {
						t.Errorf("%s entry %d (%s): body re-encode mismatch\n got % x\nwant % x",
							ent.Name(), i, e.LinkType, body, e.Tail)
					}
					var origSecondary []byte
					if e.SecondaryPath != nil {
						origSecondary = e.SecondaryPath.Raw
					}
					if !bytesEqual(secondary, origSecondary) {
						t.Errorf("%s entry %d (%s): secondary re-encode mismatch\n got % x\nwant % x",
							ent.Name(), i, e.LinkType, secondary, origSecondary)
					}
					if _, opaque := target.(linkobj.OpaqueTarget); !opaque {
						typedTargets++
					}
				}
			}
		}
	}

	if sections == 0 || totalEntries == 0 {
		t.Fatal("expected at least one LIbd section with entries in corpus")
	}
	t.Logf("LIbd coverage: %d sections, %d entries, %d typed targets, %d opaque",
		sections, totalEntries, typedTargets, totalEntries-typedTargets)
}

func bytesEqual(a, b []byte) bool {
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
