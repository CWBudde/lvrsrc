package icon

import (
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
)

func lookupFromMap(m map[string][]byte) PayloadLookup {
	return func(fourCC string) ([]byte, bool) {
		b, ok := m[fourCC]
		return b, ok
	}
}

func TestPickBestPrefersIcl8(t *testing.T) {
	lookup := lookupFromMap(map[string][]byte{
		string(MonoFourCC):   make([]byte, 128),
		string(Color4FourCC): make([]byte, 512),
		string(Color8FourCC): make([]byte, 1024),
	})
	got, ok := PickBest(codecs.Context{}, lookup)
	if !ok {
		t.Fatalf("PickBest returned !ok")
	}
	if got.FourCC != string(Color8FourCC) {
		t.Fatalf("FourCC = %q, want %q", got.FourCC, Color8FourCC)
	}
	if got.Value.BitsPerPixel != 8 {
		t.Fatalf("BitsPerPixel = %d, want 8", got.Value.BitsPerPixel)
	}
	if len(got.Value.Palette) != 256 {
		t.Fatalf("len(Palette) = %d, want 256", len(got.Value.Palette))
	}
}

func TestPickBestFallsBackToIcl4(t *testing.T) {
	lookup := lookupFromMap(map[string][]byte{
		string(MonoFourCC):   make([]byte, 128),
		string(Color4FourCC): make([]byte, 512),
	})
	got, ok := PickBest(codecs.Context{}, lookup)
	if !ok {
		t.Fatalf("PickBest returned !ok")
	}
	if got.FourCC != string(Color4FourCC) {
		t.Fatalf("FourCC = %q, want %q", got.FourCC, Color4FourCC)
	}
}

func TestPickBestFallsBackToMono(t *testing.T) {
	lookup := lookupFromMap(map[string][]byte{
		string(MonoFourCC): make([]byte, 128),
	})
	got, ok := PickBest(codecs.Context{}, lookup)
	if !ok {
		t.Fatalf("PickBest returned !ok")
	}
	if got.FourCC != string(MonoFourCC) {
		t.Fatalf("FourCC = %q, want %q", got.FourCC, MonoFourCC)
	}
	if len(got.Value.Palette) != 2 {
		t.Fatalf("len(Palette) = %d, want 2", len(got.Value.Palette))
	}
}

func TestPickBestReturnsFalseWhenNoIconPresent(t *testing.T) {
	lookup := lookupFromMap(map[string][]byte{})
	if _, ok := PickBest(codecs.Context{}, lookup); ok {
		t.Fatalf("PickBest returned ok with no icons")
	}
}

func TestPickBestSkipsWrongSize(t *testing.T) {
	// icl8 present but wrong size; should fall through to icl4.
	lookup := lookupFromMap(map[string][]byte{
		string(Color8FourCC): make([]byte, 999), // wrong
		string(Color4FourCC): make([]byte, 512), // good
	})
	got, ok := PickBest(codecs.Context{}, lookup)
	if !ok {
		t.Fatalf("PickBest returned !ok")
	}
	if got.FourCC != string(Color4FourCC) {
		t.Fatalf("FourCC = %q, want %q", got.FourCC, Color4FourCC)
	}
}
