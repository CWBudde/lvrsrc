package icon

import (
	"bytes"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
)

func TestRGBALengthMatchesImage(t *testing.T) {
	// For a 32x32 icon, RGBA() must be exactly 32*32*4 = 4096 bytes.
	v := Value{
		Width:        Width,
		Height:       Height,
		BitsPerPixel: 8,
		Pixels:       make([]byte, PixelCount),
		Palette:      paletteFor(8),
	}
	got := v.RGBA()
	if want := Width * Height * 4; len(got) != want {
		t.Fatalf("len(RGBA) = %d, want %d", len(got), want)
	}
}

func TestRGBAMonoTwoPixelsBlackAndWhite(t *testing.T) {
	// Hand-crafted pixel buffer: pixel 0 is white (index 0), pixel 1 is black
	// (index 1). RGBA byte order is R,G,B,A. Alpha is always 0xFF.
	v := Value{
		Width:        2,
		Height:       1,
		BitsPerPixel: 1,
		Pixels:       []byte{0, 1},
		Palette:      paletteFor(1),
	}
	want := []byte{
		0xFF, 0xFF, 0xFF, 0xFF, // white
		0x00, 0x00, 0x00, 0xFF, // black
	}
	if got := v.RGBA(); !bytes.Equal(got, want) {
		t.Fatalf("RGBA = %v, want %v", got, want)
	}
}

func TestRGBAIcl4PixelsMatchPalette16(t *testing.T) {
	// Two pixels: index 3 (red 0xFF0000) and index 8 (green 0x00FF00).
	v := Value{
		Width:        2,
		Height:       1,
		BitsPerPixel: 4,
		Pixels:       []byte{3, 8},
		Palette:      paletteFor(4),
	}
	want := []byte{
		0xFF, 0x00, 0x00, 0xFF, // red
		0x00, 0xFF, 0x00, 0xFF, // green
	}
	if got := v.RGBA(); !bytes.Equal(got, want) {
		t.Fatalf("RGBA = %v, want %v", got, want)
	}
}

func TestRGBAIcl8PixelsMatchPalette256(t *testing.T) {
	// Index 35 is pure red (0xFF0000) per TestPalette256SpotValues.
	v := Value{
		Width:        1,
		Height:       1,
		BitsPerPixel: 8,
		Pixels:       []byte{35},
		Palette:      paletteFor(8),
	}
	want := []byte{0xFF, 0x00, 0x00, 0xFF}
	if got := v.RGBA(); !bytes.Equal(got, want) {
		t.Fatalf("RGBA = %v, want %v", got, want)
	}
}

func TestRGBAIcl8DecodedFromSyntheticPayloadFirstPixel(t *testing.T) {
	// End-to-end: run Decode, then RGBA(), confirm the first pixel's RGBA
	// matches Palette256[payload[0]].
	payload := make([]byte, 1024)
	payload[0] = 35 // pure red
	v, err := decodeToValue(t, Color8Codec{}, payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rgba := v.RGBA()
	want := []byte{0xFF, 0x00, 0x00, 0xFF}
	if got := rgba[:4]; !bytes.Equal(got, want) {
		t.Fatalf("RGBA[0:4] = %v, want %v", got, want)
	}
}

func TestRGBAIcl8OutOfRangeIndexFallsBackToOpaqueBlack(t *testing.T) {
	// If a Pixels entry is outside the Palette (defensive — should not happen
	// with Decode output), RGBA() must still produce a well-formed
	// alpha-opaque byte run instead of panicking.
	v := Value{
		Width:        1,
		Height:       1,
		BitsPerPixel: 8,
		Pixels:       []byte{99},
		Palette:      []uint32{0xFFFF0000}, // single-entry palette
	}
	want := []byte{0x00, 0x00, 0x00, 0xFF}
	if got := v.RGBA(); !bytes.Equal(got, want) {
		t.Fatalf("RGBA = %v, want %v", got, want)
	}
}

// silence unused-import warnings if this file is the only consumer of codecs.
var _ codecs.Context
