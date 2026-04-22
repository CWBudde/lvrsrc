package rsrcwire

import "testing"

func TestParseWithOptionsLenientCollectsSecondaryHeaderMismatch(t *testing.T) {
	data := buildSyntheticRSRC(t)
	infoOffset := int(readU32BE(t, data, 16))
	data[infoOffset+8] ^= 0x01

	f, err := ParseWithOptions(data, ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	if got, want := len(f.ParseIssues), 1; got != want {
		t.Fatalf("len(ParseIssues) = %d, want %d", got, want)
	}
	if got, want := f.ParseIssues[0].Code, "header.mismatch"; got != want {
		t.Fatalf("ParseIssues[0].Code = %q, want %q", got, want)
	}
}

func TestParseWithOptionsStrictFailsOnSecondaryHeaderMismatch(t *testing.T) {
	data := buildSyntheticRSRC(t)
	infoOffset := int(readU32BE(t, data, 16))
	data[infoOffset+8] ^= 0x01

	if _, err := ParseWithOptions(data, ParseOptions{Strict: true}); err == nil {
		t.Fatal("ParseWithOptions() error = nil, want non-nil")
	}
}

func TestParseWithOptionsLenientStillFailsOnTruncation(t *testing.T) {
	data := buildSyntheticRSRC(t)
	data = data[:len(data)-1]

	if _, err := ParseWithOptions(data, ParseOptions{}); err == nil {
		t.Fatal("ParseWithOptions() error = nil, want non-nil")
	}
}

func readU32BE(t *testing.T, data []byte, off int) uint32 {
	t.Helper()
	if off+4 > len(data) {
		t.Fatalf("readU32BE(%d) out of bounds for len=%d", off, len(data))
	}

	return uint32(data[off])<<24 |
		uint32(data[off+1])<<16 |
		uint32(data[off+2])<<8 |
		uint32(data[off+3])
}
