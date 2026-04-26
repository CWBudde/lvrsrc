package fphb

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestCapability(t *testing.T) {
	c := Codec{}.Capability()
	if c.FourCC != FourCC {
		t.Fatalf("FourCC = %q, want %q", c.FourCC, FourCC)
	}
	if c.Safety != codecs.SafetyTier1 {
		t.Fatalf("Safety = %v, want SafetyTier1", c.Safety)
	}
}

func TestDecodeRejectsTruncatedPayload(t *testing.T) {
	for _, payload := range [][]byte{nil, {0, 0, 0}, {0, 0, 0, 8, 0xDE, 0xAD}} {
		if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
			t.Errorf("Decode(%d-byte payload) returned nil error", len(payload))
		}
	}
}

func TestDecodePopulatesEnvelopeAndTree(t *testing.T) {
	// Use a known-good corpus payload via a sniff test.
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	for _, e := range entries {
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		for _, block := range f.Blocks {
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				raw, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("Decode: %v", err)
				}
				v, ok := raw.(Value)
				if !ok {
					t.Fatalf("Decode returned %T, want Value", raw)
				}
				if v.Envelope.ContentLen == 0 {
					continue // empty heap is rare; skip if encountered
				}
				if len(v.Tree.Flat) == 0 {
					t.Errorf("%s FPHb has %d-byte content but empty Tree.Flat", e.Name(), v.Envelope.ContentLen)
				}
				return // one good sample is enough for this test
			}
		}
	}
	t.Skip("no FPHb sections in corpus")
}

func TestEncodeRoundTripCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalSections := 0
	totalEntries := 0
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
				totalSections++
				raw, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("%s id=%d Decode: %v", e.Name(), section.Index, err)
				}
				v := raw.(Value)
				totalEntries += len(v.Tree.Flat)
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("%s id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s FPHb id=%d round-trip mismatch (orig %d bytes, re-encoded %d bytes)",
						e.Name(), section.Index, len(section.Payload), len(back))
				}
				issues := Codec{}.Validate(codecs.Context{}, section.Payload)
				if len(issues) != 0 {
					t.Errorf("%s FPHb id=%d Validate issues: %+v", e.Name(), section.Index, issues)
				}
			}
		}
	}
	if totalSections == 0 {
		t.Skip("no FPHb sections in corpus")
	}
	t.Logf("FPHb codec round-tripped %d corpus sections (%d total tag entries)", totalSections, totalEntries)
}

func TestValidateOnTruncatedPayload(t *testing.T) {
	payload := []byte{0, 0, 0, 8, 0xDE, 0xAD}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	if len(issues) == 0 {
		t.Fatal("Validate(truncated) returned no issues")
	}
	if issues[0].Severity != validate.SeverityError {
		t.Errorf("severity = %v, want error", issues[0].Severity)
	}
}

func TestEncodeRecompressesWhenEnvelopeCacheCleared(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	for _, e := range entries {
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		for _, block := range f.Blocks {
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				raw, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("Decode: %v", err)
				}
				v := raw.(Value)
				// Drop the cached compressed bytes; Encode must
				// recompress the inflated buffer through the
				// std-library zlib writer. The result is unlikely to
				// be byte-identical to the original (zlib compression
				// level / state differs) but Decode of the re-encoded
				// payload must yield the same Content.
				v.Envelope.Compressed = nil
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("Encode after cache clear: %v", err)
				}
				envBack, err := heap.DecodeEnvelope(back)
				if err != nil {
					t.Fatalf("Decode of recompressed payload: %v", err)
				}
				if !bytes.Equal(envBack.Content, v.Envelope.Content) {
					t.Fatalf("recompressed Content drifted: %d vs %d bytes",
						len(envBack.Content), len(v.Envelope.Content))
				}
				return
			}
		}
	}
	t.Skip("no FPHb sections in corpus")
}

// TestEncodeRejectsBadInput exercises the two error branches of Encode
// (nil typed pointer + wrong concrete type) that the round-trip fixtures
// never hit.
func TestEncodeRejectsBadInput(t *testing.T) {
	if _, err := (Codec{}).Encode(codecs.Context{}, (*Value)(nil)); err == nil {
		t.Errorf("Encode(nil *Value) returned no error")
	}
	if _, err := (Codec{}).Encode(codecs.Context{}, "not a Value"); err == nil {
		t.Errorf("Encode(string) returned no error")
	}
}
