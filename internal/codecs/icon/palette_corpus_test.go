package icon

import (
	"bytes"
	"os"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestRGBAMatchesPaletteOnCorpusIcl8 spot-checks one real corpus icl8 pixel
// to be sure the ported palette + RGBA expansion agree with the on-disk
// indices. Each pixel's first four RGBA bytes must match the corresponding
// Palette256 entry (alpha 0xFF).
func TestRGBAMatchesPaletteOnCorpusIcl8(t *testing.T) {
	const fixture = "format-string.vi"
	path := corpus.Path(fixture)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("corpus fixture %s not present: %v", fixture, err)
	}

	f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}

	var icl8 []byte
	for _, block := range f.Blocks {
		if block.Type != string(Color8FourCC) {
			continue
		}
		if len(block.Sections) == 0 {
			continue
		}
		icl8 = block.Sections[0].Payload
		break
	}
	if icl8 == nil {
		t.Skipf("no icl8 section in %s", fixture)
	}
	if len(icl8) != 1024 {
		t.Fatalf("icl8 payload = %d bytes, want 1024", len(icl8))
	}

	raw, err := (Color8Codec{}).Decode(codecs.Context{}, icl8)
	if err != nil {
		t.Fatalf("decode icl8: %v", err)
	}
	v := raw.(Value)

	rgba := v.RGBA()
	if got, want := len(rgba), PixelCount*4; got != want {
		t.Fatalf("len(RGBA) = %d, want %d", got, want)
	}

	// Spot-check pixel 0 and the last pixel. Both must match the palette
	// entry for their on-disk index, with alpha 0xFF.
	for _, pixelIdx := range []int{0, PixelCount - 1} {
		want := expectedRGBA(Palette256[icl8[pixelIdx]])
		got := rgba[pixelIdx*4 : pixelIdx*4+4]
		if !bytes.Equal(got, want) {
			t.Errorf("pixel %d (index %d): RGBA = %v, want %v",
				pixelIdx, icl8[pixelIdx], got, want)
		}
	}
}

func expectedRGBA(argb uint32) []byte {
	return []byte{
		byte(argb >> 16),
		byte(argb >> 8),
		byte(argb),
		0xFF,
	}
}
