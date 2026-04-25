package fphb

import (
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// FuzzDecode feeds arbitrary bytes to Codec.Decode. The contract is
// the same as the envelope-level fuzz: Decode must never panic, and
// must either succeed or return an error.
func FuzzDecode(f *testing.F) {
	// Seed with something that decodes cleanly so the fuzzer has
	// coverage of both the envelope and walker paths.
	f.Add(buildSeed([]byte{0x04, 50})) // single TagLeaf rawTagID=50, no content
	// Plus a few hand-crafted truncations.
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{0, 0, 0, 8, 0xDE, 0xAD})
	// Also add real corpus payloads when present.
	for _, seed := range corpusSeeds() {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Decode panicked on payload len %d: %v", len(payload), r)
			}
		}()
		_, _ = Codec{}.Decode(codecs.Context{}, payload)
	})
}

// FuzzValidate feeds arbitrary bytes to Codec.Validate. Like Decode it
// must never panic; the issue list is allowed to be empty or non-empty.
func FuzzValidate(f *testing.F) {
	f.Add(buildSeed([]byte{0x04, 50}))
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 5, 0xFF})
	for _, seed := range corpusSeeds() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Validate panicked on payload len %d: %v", len(payload), r)
			}
		}()
		_ = Codec{}.Validate(codecs.Context{}, payload)
	})
}

// buildSeed wraps the given tag-stream content into a heap envelope
// payload. Mirrors the helper in envelope_test.go without sharing
// state across packages.
func buildSeed(content []byte) []byte {
	inflated := make([]byte, 4+len(content))
	binary.BigEndian.PutUint32(inflated[:4], uint32(len(content)))
	copy(inflated[4:], content)

	var buf zbuf
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(inflated); err != nil {
		_ = w.Close()
		return nil
	}
	if err := w.Close(); err != nil {
		return nil
	}
	out := make([]byte, 4+len(buf.b))
	binary.BigEndian.PutUint32(out[:4], uint32(len(inflated)))
	copy(out[4:], buf.b)
	return out
}

// corpusSeeds returns up to 8 real FPHb payloads from the corpus, or
// nil when the corpus directory is not present (e.g. on a clean clone
// in CI).
func corpusSeeds() [][]byte {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		return nil
	}
	var seeds [][]byte
	for _, e := range entries {
		if len(seeds) >= 8 {
			break
		}
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
		for _, block := range f.Blocks {
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				dup := make([]byte, len(section.Payload))
				copy(dup, section.Payload)
				seeds = append(seeds, dup)
				break
			}
			break
		}
	}
	return seeds
}

type zbuf struct{ b []byte }

func (z *zbuf) Write(p []byte) (int, error) {
	z.b = append(z.b, p...)
	return len(p), nil
}
