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

// TestParseWithOptionsLenientCollectsBadNameOffset injects an
// out-of-range section name offset and verifies the parser surfaces it
// as a ParseIssue rather than aborting (lenient mode) and as an error
// in strict mode.
func TestParseWithOptionsLenientCollectsBadNameOffset(t *testing.T) {
	data := buildSyntheticRSRC(t)
	infoOffset := int(readU32BE(t, data, 16))
	// First section name offset lives 4 bytes into the first sectionStart
	// record, which sits at offset infoOffset + listHeaderSize + blockInfoSize + blockHeaderSize.
	// Layout (parser_test.go's buildSyntheticRSRC):
	//   secondary header (32) + list header (20) + block count (4) +
	//   block header (12) → 68 bytes before first sectionStart record.
	const sectionStartsRelInfo = 32 + 20 + 4 + 12
	firstSection := infoOffset + sectionStartsRelInfo
	// 0xffffffff is the noNameOffset sentinel and would be silently
	// ignored. Use a large but non-sentinel value so it lands past the
	// info section end and triggers the "name offset beyond info
	// section" guard.
	data[firstSection+4] = 0x0f
	data[firstSection+5] = 0xff
	data[firstSection+6] = 0xff
	data[firstSection+7] = 0xff

	f, err := ParseWithOptions(data, ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions(lenient) error = %v", err)
	}
	found := false
	for _, issue := range f.ParseIssues {
		if issue.Code == "section.name_offset.invalid" {
			found = true
		}
	}
	if !found {
		t.Errorf("ParseIssues missing section.name_offset.invalid: %+v", f.ParseIssues)
	}

	if _, err := ParseWithOptions(data, ParseOptions{Strict: true}); err == nil {
		t.Errorf("ParseWithOptions(strict) returned no error for bad name offset")
	}
}
