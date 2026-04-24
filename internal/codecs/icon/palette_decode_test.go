package icon

import (
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
)

func TestDecodePopulatesPaletteForMono(t *testing.T) {
	payload := make([]byte, 128)
	v, err := decodeToValue(t, MonoCodec{}, payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := len(v.Palette), 2; got != want {
		t.Fatalf("len(Palette) = %d, want %d", got, want)
	}
	if v.Palette[0] != Palette2[0] || v.Palette[1] != Palette2[1] {
		t.Errorf("Palette = %v, want %v", v.Palette, Palette2)
	}
}

func TestDecodePopulatesPaletteForIcl4(t *testing.T) {
	payload := make([]byte, 512)
	v, err := decodeToValue(t, Color4Codec{}, payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := len(v.Palette), 16; got != want {
		t.Fatalf("len(Palette) = %d, want %d", got, want)
	}
	// Random-access spot checks are covered by TestPalette16SpotValues; here
	// we just need to be sure Decode wired the right table.
	if v.Palette[0] != Palette16[0] || v.Palette[15] != Palette16[15] {
		t.Errorf("Palette = %v, want Palette16", v.Palette)
	}
}

func TestDecodePopulatesPaletteForIcl8(t *testing.T) {
	payload := make([]byte, 1024)
	v, err := decodeToValue(t, Color8Codec{}, payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := len(v.Palette), 256; got != want {
		t.Fatalf("len(Palette) = %d, want %d", got, want)
	}
	if v.Palette[0] != Palette256[0] || v.Palette[255] != Palette256[255] {
		t.Errorf("Palette boundaries = [%#x, %#x], want [%#x, %#x]",
			v.Palette[0], v.Palette[255], Palette256[0], Palette256[255])
	}
}

// decodeToValue calls Decode and asserts the result is a Value.
func decodeToValue(t *testing.T, codec codecs.ResourceCodec, payload []byte) (Value, error) {
	t.Helper()
	raw, err := codec.Decode(codecs.Context{}, payload)
	if err != nil {
		return Value{}, err
	}
	v, ok := raw.(Value)
	if !ok {
		t.Fatalf("Decode returned %T, want Value", raw)
	}
	return v, nil
}
