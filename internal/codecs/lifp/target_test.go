package lifp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/lifp"
	"github.com/CWBudde/lvrsrc/internal/codecs/linkobj"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestEntryTarget_Corpus walks every LIfp section in the corpus and
// confirms that each entry's Target() decodes into a known LinkTarget
// (either typed or OpaqueTarget). The decoded target is then re-encoded
// and the result is checked against the entry's original Tail +
// SecondaryPath bytes — so this is the strongest round-trip guarantee
// the typed surface offers.
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
			continue // not all corpus files are RSRC
		}
		ctx := codecs.Context{FileVersion: f.Header.FormatVersion, Kind: f.Kind}
		for _, blk := range f.Blocks {
			if blk.Type != "LIfp" {
				continue
			}
			for _, sec := range blk.Sections {
				out, err := (lifp.Codec{}).Decode(ctx, sec.Payload)
				if err != nil {
					t.Errorf("%s: LIfp decode: %v", ent.Name(), err)
					continue
				}
				v := out.(lifp.Value)
				sections++
				for i, e := range v.Entries {
					totalEntries++
					target, err := e.Target()
					if err != nil {
						t.Errorf("%s entry %d (%s): Target: %v", ent.Name(), i, e.LinkType, err)
						continue
					}
					if target.Ident() != e.LinkType {
						t.Errorf("%s entry %d: target ident %q != entry link type %q",
							ent.Name(), i, target.Ident(), e.LinkType)
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
		t.Fatal("expected at least one LIfp section with entries in corpus")
	}
	t.Logf("LIfp coverage: %d sections, %d entries, %d typed targets, %d opaque",
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
