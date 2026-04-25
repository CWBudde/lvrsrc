package pthx

import (
	"bytes"
	"testing"
)

func TestDecodePTH0SimplePath(t *testing.T) {
	// PTH0 ident + totlen=18 + tpval=0 + count=2 + Pascal-string components
	payload := []byte{
		'P', 'T', 'H', '0',
		0, 0, 0, 18, // totlen = 4 (header tpval+count) + 1+5 + 1+7 = 18
		0, 0, // tpval = 0
		0, 2, // count = 2
		5, 'h', 'e', 'l', 'l', 'o',
		7, 'g', 'o', 'o', 'd', 'b', 'y', 'e',
	}
	v, n, err := Decode(payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if n != len(payload) {
		t.Errorf("consumed = %d, want %d", n, len(payload))
	}
	if v.Ident != "PTH0" {
		t.Errorf("Ident = %q, want PTH0", v.Ident)
	}
	if !v.IsPTH0() {
		t.Errorf("IsPTH0() = false")
	}
	if v.TPVal != 0 {
		t.Errorf("TPVal = %d, want 0", v.TPVal)
	}
	if len(v.Components) != 2 {
		t.Fatalf("len(Components) = %d, want 2", len(v.Components))
	}
	if string(v.Components[0]) != "hello" || string(v.Components[1]) != "goodbye" {
		t.Errorf("Components = %v", v.Components)
	}
	if v.ZeroFill {
		t.Errorf("ZeroFill = true, want false")
	}
}

func TestDecodePTH0EmptyPath(t *testing.T) {
	// Empty PTH0: ident + totlen=4 + tpval=0 + count=0 (no components)
	payload := []byte{'P', 'T', 'H', '0', 0, 0, 0, 4, 0, 0, 0, 0}
	v, n, err := Decode(payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if n != 12 {
		t.Errorf("consumed = %d, want 12", n)
	}
	if len(v.Components) != 0 {
		t.Errorf("Components = %v, want empty", v.Components)
	}
	if v.ZeroFill {
		t.Errorf("ZeroFill = true on legitimate empty path")
	}
}

func TestDecodePTH0ZeroFillPhony(t *testing.T) {
	// LV "phony" form: 4 bytes ident + 4 bytes zero totlen, with no further
	// content. pylabview's canZeroFill case.
	payload := []byte{'P', 'T', 'H', '0', 0, 0, 0, 0}
	v, n, err := Decode(payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if n != 8 {
		t.Errorf("consumed = %d, want 8", n)
	}
	if !v.ZeroFill {
		t.Error("ZeroFill = false, want true")
	}
	if !v.IsPhony() {
		t.Error("IsPhony() = false, want true")
	}
	if len(v.Components) != 0 || v.TPVal != 0 {
		t.Errorf("zero-fill should yield empty components and TPVal=0; got %+v", v)
	}
}

func TestDecodePTH1AbsolutePath(t *testing.T) {
	// PTH1 ident + totlen + tpident "abs " + 2 components with 2-byte lengths
	payload := []byte{
		'P', 'T', 'H', '1',
		0, 0, 0, 15,
		'a', 'b', 's', ' ',
		0, 2, 'h', 'i',
		0, 5, 'w', 'o', 'r', 'l', 'd',
	}
	v, n, err := Decode(payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if n != len(payload) {
		t.Errorf("consumed = %d, want %d", n, len(payload))
	}
	if v.Ident != "PTH1" {
		t.Errorf("Ident = %q, want PTH1", v.Ident)
	}
	if !v.IsPTH1() {
		t.Errorf("IsPTH1() = false")
	}
	if v.TPIdent != "abs " {
		t.Errorf("TPIdent = %q, want %q", v.TPIdent, "abs ")
	}
	if !v.IsAbsolute() {
		t.Errorf("IsAbsolute() = false")
	}
	if len(v.Components) != 2 || string(v.Components[0]) != "hi" || string(v.Components[1]) != "world" {
		t.Errorf("Components = %v", v.Components)
	}
}

func TestDecodePTH1RelPathHelpers(t *testing.T) {
	for _, tc := range []struct {
		tpident  string
		isAbs    bool
		isRel    bool
		isUNC    bool
		isNotPth bool
	}{
		{"abs ", true, false, false, false},
		{"rel ", false, true, false, false},
		{"unc ", false, false, true, false},
		{"!pth", false, false, false, true},
	} {
		t.Run(tc.tpident, func(t *testing.T) {
			payload := []byte{'P', 'T', 'H', '1', 0, 0, 0, 4, tc.tpident[0], tc.tpident[1], tc.tpident[2], tc.tpident[3]}
			v, _, err := Decode(payload)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if v.IsAbsolute() != tc.isAbs {
				t.Errorf("IsAbsolute() = %v, want %v", v.IsAbsolute(), tc.isAbs)
			}
			if v.IsRelative() != tc.isRel {
				t.Errorf("IsRelative() = %v, want %v", v.IsRelative(), tc.isRel)
			}
			if v.IsUNC() != tc.isUNC {
				t.Errorf("IsUNC() = %v, want %v", v.IsUNC(), tc.isUNC)
			}
			if v.IsNotAPath() != tc.isNotPth {
				t.Errorf("IsNotAPath() = %v, want %v", v.IsNotAPath(), tc.isNotPth)
			}
		})
	}
}

func TestDecodePTH2AcceptedAsPTH1Variant(t *testing.T) {
	payload := []byte{
		'P', 'T', 'H', '2',
		0, 0, 0, 6,
		'r', 'e', 'l', ' ',
		0, 0,
	}
	v, _, err := Decode(payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	// Wait — PTH2 with 6-byte totlen = 4 (tpident) + 2 (one zero-length component? no, 2 = 2-byte length only)
	// Adjust: include a 0-length component
	if v.Ident != "PTH2" {
		t.Errorf("Ident = %q, want PTH2", v.Ident)
	}
	if !v.IsPTH1() {
		t.Errorf("IsPTH1() should accept PTH2 variant")
	}
}

func TestDecodeRejectsUnknownIdent(t *testing.T) {
	payload := []byte{'X', 'X', 'X', 'X', 0, 0, 0, 0}
	if _, _, err := Decode(payload); err == nil {
		t.Fatal("Decode of unknown ident returned nil error")
	}
}

func TestDecodeRejectsTruncatedHeader(t *testing.T) {
	for _, payload := range [][]byte{nil, {1, 2, 3}, {'P', 'T', 'H', '0'}} {
		if _, _, err := Decode(payload); err == nil {
			t.Errorf("Decode(%d bytes) returned nil error", len(payload))
		}
	}
}

func TestEncodeRoundTripPTH0(t *testing.T) {
	// totlen counts: tpval(2) + count(2) + (1+3) + (1+2) = 11
	original := []byte{
		'P', 'T', 'H', '0',
		0, 0, 0, 11,
		0, 1, // tpval = 1
		0, 2,
		3, 'a', 'b', 'c',
		2, 'x', 'y',
	}
	v, _, err := Decode(original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestEncodeRoundTripPTH0ZeroFill(t *testing.T) {
	original := []byte{'P', 'T', 'H', '0', 0, 0, 0, 0}
	v, _, err := Decode(original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestEncodeRoundTripPTH1(t *testing.T) {
	original := []byte{
		'P', 'T', 'H', '1',
		0, 0, 0, 15,
		'a', 'b', 's', ' ',
		0, 2, 'h', 'i',
		0, 5, 'w', 'o', 'r', 'l', 'd',
	}
	v, _, err := Decode(original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestEncodeRejectsLongComponentForPTH0(t *testing.T) {
	v := Value{
		Ident:      "PTH0",
		Components: [][]byte{make([]byte, 256)},
	}
	if _, err := Encode(v); err == nil {
		t.Fatal("Encode of 256-byte PTH0 component returned nil error")
	}
}
