package heap

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// buildEnvelopeBytes synthesises a heap envelope payload around the given
// inner content bytes (the tag-stream). Layout:
//
//	[u32 declared_size BE] [ zlib-compressed { [u32 content_len BE] + content } ]
//
// where declared_size = 4 + len(content).
func buildEnvelopeBytes(t *testing.T, content []byte) []byte {
	t.Helper()
	inflated := make([]byte, 4+len(content))
	binary.BigEndian.PutUint32(inflated[:4], uint32(len(content)))
	copy(inflated[4:], content)

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(inflated); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	out := make([]byte, 4+buf.Len())
	binary.BigEndian.PutUint32(out[:4], uint32(len(inflated)))
	copy(out[4:], buf.Bytes())
	return out
}

func TestDecodeEnvelopeBasicShape(t *testing.T) {
	content := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	payload := buildEnvelopeBytes(t, content)

	env, err := DecodeEnvelope(payload)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	if env.DeclaredSize != 4+uint32(len(content)) {
		t.Errorf("DeclaredSize = %d, want %d", env.DeclaredSize, 4+len(content))
	}
	if env.ContentLen != uint32(len(content)) {
		t.Errorf("ContentLen = %d, want %d", env.ContentLen, len(content))
	}
	if !bytes.Equal(env.Content, content) {
		t.Errorf("Content = %x, want %x", env.Content, content)
	}
}

func TestDecodeEnvelopeRejectsTooShortForOuterHeader(t *testing.T) {
	for _, payload := range [][]byte{nil, {0}, {0, 0, 0}} {
		if _, err := DecodeEnvelope(payload); err == nil {
			t.Errorf("DecodeEnvelope(%d bytes) returned nil error", len(payload))
		}
	}
}

func TestDecodeEnvelopeRejectsBadZlib(t *testing.T) {
	// Outer header claims 32 inflated bytes but the zlib body is garbage.
	payload := []byte{0, 0, 0, 32, 0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}
	if _, err := DecodeEnvelope(payload); err == nil {
		t.Fatal("DecodeEnvelope of garbage zlib returned nil error")
	}
}

func TestDecodeEnvelopeRejectsDeclaredSizeMismatch(t *testing.T) {
	// Build a normal envelope, then mutate the outer declared size to lie
	// about the inflated length.
	content := []byte{0xAA, 0xBB}
	payload := buildEnvelopeBytes(t, content)
	binary.BigEndian.PutUint32(payload[:4], 999)
	if _, err := DecodeEnvelope(payload); err == nil {
		t.Fatal("DecodeEnvelope of mismatched declared size returned nil error")
	}
}

func TestDecodeEnvelopeRejectsInnerLengthMismatch(t *testing.T) {
	// Build an inflated buffer where the inner content_len lies. We have to
	// write the lying buffer ourselves so the outer declared_size still
	// matches the inflated length but the inner content_len is wrong.
	content := []byte{0xAA, 0xBB}
	inflated := make([]byte, 4+len(content))
	binary.BigEndian.PutUint32(inflated[:4], 99) // lie: claim 99-byte content
	copy(inflated[4:], content)

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(inflated); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	payload := make([]byte, 4+buf.Len())
	binary.BigEndian.PutUint32(payload[:4], uint32(len(inflated)))
	copy(payload[4:], buf.Bytes())

	if _, err := DecodeEnvelope(payload); err == nil {
		t.Fatal("DecodeEnvelope of inner-size mismatch returned nil error")
	}
}

func TestEncodeEnvelopeRoundTrip(t *testing.T) {
	original := buildEnvelopeBytes(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05})
	env, err := DecodeEnvelope(original)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	back, err := EncodeEnvelope(env)
	if err != nil {
		t.Fatalf("EncodeEnvelope: %v", err)
	}
	// Re-decode `back` to be sure we round-trip semantically. Byte
	// equality is not guaranteed because zlib output depends on
	// compression level / flush state — pylabview itself accepts that
	// trade-off when re-emitting.
	env2, err := DecodeEnvelope(back)
	if err != nil {
		t.Fatalf("DecodeEnvelope(re-encoded): %v", err)
	}
	if !bytes.Equal(env2.Content, env.Content) {
		t.Fatalf("round-trip Content drift: %x vs %x", env2.Content, env.Content)
	}
	if env2.ContentLen != env.ContentLen {
		t.Errorf("ContentLen drift: %d vs %d", env2.ContentLen, env.ContentLen)
	}
}

func TestEncodeReusesCompressedWhenAvailable(t *testing.T) {
	// When env.Compressed is already populated and matches Content, Encode
	// should reuse those bytes verbatim — yielding byte-identical output.
	original := buildEnvelopeBytes(t, []byte{0x10, 0x20, 0x30})
	env, err := DecodeEnvelope(original)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	back, err := EncodeEnvelope(env)
	if err != nil {
		t.Fatalf("EncodeEnvelope: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original even though Compressed was preserved:\n got %x\nwant %x", back, original)
	}
}

func TestEncodeRecompressesWhenContentEdited(t *testing.T) {
	original := buildEnvelopeBytes(t, []byte{0x10, 0x20, 0x30})
	env, err := DecodeEnvelope(original)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	// Caller mutates the content. Compressed cache no longer matches —
	// Encode must re-deflate.
	env.Content = append(env.Content, 0xFF)
	env.ContentLen++
	env.DeclaredSize++
	back, err := EncodeEnvelope(env)
	if err != nil {
		t.Fatalf("EncodeEnvelope: %v", err)
	}
	env2, err := DecodeEnvelope(back)
	if err != nil {
		t.Fatalf("DecodeEnvelope(re-encoded): %v", err)
	}
	if len(env2.Content) != len(env.Content) {
		t.Errorf("re-encoded Content len = %d, want %d", len(env2.Content), len(env.Content))
	}
}

func TestDecodeEnvelopeOnCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalFP, totalBD := 0, 0
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
				if env.ContentLen != uint32(len(env.Content)) {
					t.Errorf("%s %s id=%d ContentLen %d != len(Content) %d",
						e.Name(), block.Type, section.Index, env.ContentLen, len(env.Content))
				}
				if env.DeclaredSize != 4+env.ContentLen {
					t.Errorf("%s %s id=%d DeclaredSize %d != 4 + ContentLen %d",
						e.Name(), block.Type, section.Index, env.DeclaredSize, env.ContentLen)
				}
				if block.Type == "FPHb" {
					totalFP++
				} else {
					totalBD++
				}
			}
		}
	}
	if totalFP+totalBD == 0 {
		t.Skip("no heap sections in corpus")
	}
	t.Logf("decoded %d FPHb + %d BDHb envelope(s) cleanly", totalFP, totalBD)
}

func TestEncodeRoundTripsCorpusSemanticEquivalent(t *testing.T) {
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
		f, _ := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		for _, block := range f.Blocks {
			if block.Type != "FPHb" && block.Type != "BDHb" {
				continue
			}
			for _, section := range block.Sections {
				env, err := DecodeEnvelope(section.Payload)
				if err != nil {
					continue
				}
				back, err := EncodeEnvelope(env)
				if err != nil {
					t.Errorf("%s EncodeEnvelope: %v", e.Name(), err)
					continue
				}
				// Compressed-cache reuse path: re-encoding without
				// touching Content must yield byte-identical output.
				if !bytes.Equal(back, section.Payload) {
					t.Errorf("%s %s id=%d cached re-encode drifted",
						e.Name(), block.Type, section.Index)
					continue
				}
				exercised++
			}
		}
	}
	if exercised == 0 {
		t.Skip("no heap sections in corpus")
	}
	t.Logf("byte-for-byte round-tripped %d corpus heap sections via Compressed cache", exercised)
}
