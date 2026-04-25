package lvdiff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestFPHbStructuralDiffIgnoresCompressedAndRedundantTreeFields exercises
// the FPHb decoded differ on a real corpus VI compared against itself.
//
// Phase 9.4 wired fphb.Codec into the registry differ; Phase 9.5
// promises a *structural* differ that does not double-count nodes via
// Tree.Roots and does not produce noise from the Envelope.Compressed
// byte cache (which is meaningful only as a round-trip optimisation).
//
// The test asserts:
//  1. Identical files produce zero decoded diff items for FPHb.
//  2. No reported diff path ever mentions the Compressed cache, the
//     Roots tree projection, or the Children index slice — those are
//     redundant with Envelope.Content / Tree.Flat / Parent and would
//     only generate noise.
func TestFPHbStructuralDiffIgnoresCompressedAndRedundantTreeFields(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	exercised := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		path := filepath.Join(corpus.Dir(), e.Name())
		a, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("open %s as a: %v", e.Name(), err)
		}
		if !hasBlock(a, "FPHb") {
			continue
		}
		b, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("open %s as b: %v", e.Name(), err)
		}

		d := Files(a, b)
		if d == nil {
			t.Fatalf("%s: Files returned nil", e.Name())
		}
		for _, item := range d.Items {
			if item.Kind != KindDecoded {
				continue
			}
			if !strings.HasPrefix(item.Path, "blocks.FPHb/") {
				continue
			}
			t.Errorf("%s: identical file produced FPHb decoded diff: %s",
				e.Name(), item.Path)
		}
		exercised++
	}
	if exercised == 0 {
		t.Skip("no FPHb-bearing corpus files exercised")
	}

	// Sanity check on the suppression semantics: even when content
	// changes legitimately, the structural differ must not surface the
	// noisy fields. Use a synthetic lvrsrc.File pair with two distinct
	// FPHb payloads taken from two different corpus files (so the
	// content really does differ).
	pair := pickTwoDistinctFPHbFiles(t)
	if pair[0] == nil || pair[1] == nil {
		t.Skip("need two distinct FPHb-bearing corpus files for noise suppression check")
	}
	d := Files(pair[0], pair[1])
	for _, item := range d.Items {
		if item.Kind != KindDecoded {
			continue
		}
		for _, banned := range []string{
			".Envelope.Compressed",
			".Tree.Roots",
			".Children",
		} {
			if strings.Contains(item.Path, banned) {
				t.Errorf("FPHb decoded diff path %q contains banned redundant field %q",
					item.Path, banned)
			}
		}
	}
}

func hasBlock(f *lvrsrc.File, fourCC string) bool {
	if f == nil {
		return false
	}
	for _, b := range f.Blocks {
		if b.Type == fourCC && len(b.Sections) > 0 {
			return true
		}
	}
	return false
}

func pickTwoDistinctFPHbFiles(t *testing.T) [2]*lvrsrc.File {
	t.Helper()
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		return [2]*lvrsrc.File{}
	}
	var picked [2]*lvrsrc.File
	idx := 0
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
		if !hasBlock(f, "FPHb") {
			continue
		}
		picked[idx] = f
		idx++
		if idx == 2 {
			break
		}
	}
	return picked
}
