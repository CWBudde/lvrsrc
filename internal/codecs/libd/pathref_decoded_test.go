package libd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestPathRefDecodedAcrossCorpus mirrors the LIfp test: every LIbd
// PrimaryPath in the corpus must decode through internal/codecs/pthx.
func TestPathRefDecodedAcrossCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalPaths := 0
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
					continue
				}
				v, ok := raw.(Value)
				if !ok {
					continue
				}
				for i, entry := range v.Entries {
					if len(entry.PrimaryPath.Raw) == 0 {
						continue
					}
					totalPaths++
					if _, err := entry.PrimaryPath.Decoded(); err != nil {
						t.Errorf("%s entry %d primary path Decoded(): %v", e.Name(), i, err)
					}
					if entry.SecondaryPath != nil && len(entry.SecondaryPath.Raw) > 0 {
						totalPaths++
						if _, err := entry.SecondaryPath.Decoded(); err != nil {
							t.Errorf("%s entry %d secondary path Decoded(): %v", e.Name(), i, err)
						}
					}
				}
			}
		}
	}
	if totalPaths == 0 {
		t.Skip("no LIbd paths exercised")
	}
	t.Logf("decoded %d LIbd path(s) through pthx", totalPaths)
}
