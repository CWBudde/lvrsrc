package linkobj

import (
	"bytes"
	"reflect"
	"testing"
)

// TDCC sample from testdata/corpus/reference-find-by-id.vi LIfp entry 1.
// Tail (23 bytes) covers OffsetLinkSaveInfo (linkSaveFlag, TDIdx, vilinkref,
// typedLinkFlags, offsetCount, offsets); ViLSPathRef is the SecondaryPath
// the surrounding codec extracted as 8-byte PTH0 declaredLen=0.
var tdccCorpusBody = []byte{
	0x00, 0x00, 0x00, 0x00, // linkSaveFlag = 0
	0x00, 0x00, // TDIdx u2p2 = 0
	0x42,                   // vilinkref flagBt = 0x42
	0x00, 0x00, 0x08, 0x00, // typedLinkFlags = 0x800
	0x00, 0x00, 0x00, 0x02, // offsetCount = 2
	0x00, 0x00, 0x01, 0x3f, // offset[0] = 0x13f
	0x00, 0x00, 0x01, 0xd3, // offset[1] = 0x1d3
}

var tdccCorpusSecondary = []byte{
	0x50, 0x54, 0x48, 0x30, 0x00, 0x00, 0x00, 0x00, // PTH0 declaredLen=0
}

func TestLookupKind_KnownIdents(t *testing.T) {
	tests := map[string]LinkKind{
		"TDCC": KindTypeDefToCCLink,
		"VILB": KindVIToLib,
		"IUVI": KindIUseToVILink,
		"VICC": KindVIToCCLink,
		"VIMS": KindVIToMSLink,
		"DNDA": KindHeapToAssembly,
		"DNVA": KindVIToAssembly,
		"VIFl": KindVIToFileLink,
		"IVOV": KindInstanceVIToOwnerVI,
	}
	for ident, want := range tests {
		if got := LookupKind(ident); got != want {
			t.Errorf("LookupKind(%q) = %v, want %v", ident, got, want)
		}
	}
}

func TestLookupKind_Unknown(t *testing.T) {
	if got := LookupKind("ZZZZ"); got != KindUnknown {
		t.Errorf("LookupKind(\"ZZZZ\") = %v, want KindUnknown", got)
	}
}

func TestCanonicalIdent(t *testing.T) {
	if got := CanonicalIdent(KindTypeDefToCCLink); got != "TDCC" {
		t.Errorf("CanonicalIdent(KindTypeDefToCCLink) = %q, want TDCC", got)
	}
	if got := CanonicalIdent(KindUnknown); got != "" {
		t.Errorf("CanonicalIdent(KindUnknown) = %q, want \"\"", got)
	}
}

func TestDescription(t *testing.T) {
	if got := Description(KindTypeDefToCCLink); got != "TypeDef → CustCtl" {
		t.Errorf("Description(KindTypeDefToCCLink) = %q", got)
	}
	if got := Description(KindUnknown); got != "unknown link" {
		t.Errorf("Description(KindUnknown) = %q", got)
	}
}

func TestTDCC_RoundTrip_Corpus(t *testing.T) {
	target, err := Decode("TDCC", tdccCorpusBody, tdccCorpusSecondary)
	if err != nil {
		t.Fatalf("Decode TDCC: %v", err)
	}
	tdcc, ok := target.(TypeDefToCCLink)
	if !ok {
		t.Fatalf("Decode returned %T, want TypeDefToCCLink", target)
	}
	if tdcc.Kind() != KindTypeDefToCCLink {
		t.Errorf("Kind() = %v, want KindTypeDefToCCLink", tdcc.Kind())
	}
	if tdcc.Ident() != "TDCC" {
		t.Errorf("Ident() = %q, want TDCC", tdcc.Ident())
	}
	if tdcc.LinkSaveFlag != 0 {
		t.Errorf("LinkSaveFlag = %d, want 0", tdcc.LinkSaveFlag)
	}
	if tdcc.TypeDescID != 0 {
		t.Errorf("TypeDescID = %d, want 0", tdcc.TypeDescID)
	}
	if tdcc.TypeDescIDWide {
		t.Errorf("TypeDescIDWide = true, want false")
	}
	if tdcc.VILinkRef.FlagBt != 0x42 {
		t.Errorf("VILinkRef.FlagBt = %#x, want 0x42", tdcc.VILinkRef.FlagBt)
	}
	if tdcc.VILinkRef.HasExplicit {
		t.Errorf("VILinkRef.HasExplicit = true, want false")
	}
	if tdcc.TypedLinkFlags != 0x800 {
		t.Errorf("TypedLinkFlags = %#x, want 0x800", tdcc.TypedLinkFlags)
	}
	if !reflect.DeepEqual(tdcc.Offsets, []uint32{0x13f, 0x1d3}) {
		t.Errorf("Offsets = %v, want [0x13f 0x1d3]", tdcc.Offsets)
	}
	if !bytes.Equal(tdcc.ViLSPathRef, tdccCorpusSecondary) {
		t.Errorf("ViLSPathRef = % x, want % x", tdcc.ViLSPathRef, tdccCorpusSecondary)
	}

	body, secondary, err := Encode(tdcc)
	if err != nil {
		t.Fatalf("Encode TDCC: %v", err)
	}
	if !bytes.Equal(body, tdccCorpusBody) {
		t.Errorf("body mismatch:\n got % x\nwant % x", body, tdccCorpusBody)
	}
	if !bytes.Equal(secondary, tdccCorpusSecondary) {
		t.Errorf("secondary mismatch:\n got % x\nwant % x", secondary, tdccCorpusSecondary)
	}
}

func TestTDCC_19Byte_OneOffset(t *testing.T) {
	// Matches reference-find-by-id.vi LIfp entry 3.
	body := []byte{
		0x00, 0x00, 0x00, 0x00, // linkSaveFlag
		0x00, 0x00, // TDIdx
		0x42,                   // vilinkref flagBt
		0x00, 0x00, 0x00, 0x00, // typedLinkFlags
		0x00, 0x00, 0x00, 0x01, // offsetCount = 1
		0x00, 0x00, 0x01, 0x83, // offset[0]
	}
	target, err := Decode("TDCC", body, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	tdcc := target.(TypeDefToCCLink)
	if !reflect.DeepEqual(tdcc.Offsets, []uint32{0x183}) {
		t.Errorf("Offsets = %v, want [0x183]", tdcc.Offsets)
	}
	if tdcc.ViLSPathRef != nil {
		t.Errorf("ViLSPathRef = %v, want nil (no secondary)", tdcc.ViLSPathRef)
	}
	body2, secondary2, err := Encode(tdcc)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(body, body2) {
		t.Errorf("body roundtrip mismatch:\n got % x\nwant % x", body2, body)
	}
	if secondary2 != nil {
		t.Errorf("secondary roundtrip = % x, want nil", secondary2)
	}
}

func TestVILB_RoundTrip_Corpus(t *testing.T) {
	// Trailing 4 bytes of config-data.ctl LIvi LVCC entry: linkSaveFlag = 2.
	body := []byte{0x00, 0x00, 0x00, 0x02}
	target, err := Decode("VILB", body, nil)
	if err != nil {
		t.Fatalf("Decode VILB: %v", err)
	}
	vilb, ok := target.(VIToLib)
	if !ok {
		t.Fatalf("Decode returned %T, want VIToLib", target)
	}
	if vilb.LinkSaveFlag != 2 {
		t.Errorf("LinkSaveFlag = %d, want 2", vilb.LinkSaveFlag)
	}
	if vilb.Kind() != KindVIToLib {
		t.Errorf("Kind() = %v, want KindVIToLib", vilb.Kind())
	}

	body2, secondary2, err := Encode(vilb)
	if err != nil {
		t.Fatalf("Encode VILB: %v", err)
	}
	if !bytes.Equal(body, body2) {
		t.Errorf("body mismatch: got % x, want % x", body2, body)
	}
	if secondary2 != nil {
		t.Errorf("VILB secondary should be nil, got % x", secondary2)
	}
}

func TestVILB_RejectsSecondary(t *testing.T) {
	body := []byte{0x00, 0x00, 0x00, 0x00}
	if _, err := Decode("VILB", body, []byte{0x50, 0x54, 0x48, 0x30, 0, 0, 0, 0}); err == nil {
		t.Error("Decode VILB with secondary path should error")
	}
}

func TestOpaque_RoundTrip(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	secondary := []byte{0x50, 0x54, 0x48, 0x30, 0x00, 0x00, 0x00, 0x00}
	target, err := Decode("ZZZZ", body, secondary)
	if err != nil {
		t.Fatalf("Decode unknown: %v", err)
	}
	if target.Kind() != KindUnknown {
		t.Errorf("Kind() = %v, want KindUnknown", target.Kind())
	}
	if target.Ident() != "ZZZZ" {
		t.Errorf("Ident() = %q, want ZZZZ", target.Ident())
	}
	body2, secondary2, err := Encode(target)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(body, body2) {
		t.Errorf("body mismatch")
	}
	if !bytes.Equal(secondary, secondary2) {
		t.Errorf("secondary mismatch")
	}
}

func TestVarsizeU2p2(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		val   uint32
		size  int
	}{
		{"narrow zero", []byte{0x00, 0x00}, 0, 2},
		{"narrow max", []byte{0x7f, 0xff}, 0x7fff, 2},
		{"wide min", []byte{0x80, 0x00, 0x00, 0x00}, 0, 4},
		{"wide arbitrary", []byte{0x80, 0x01, 0x02, 0x03}, (1 << 16) | 0x0203, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, n, err := readVarSizeU2p2(tt.bytes)
			if err != nil {
				t.Fatalf("readVarSizeU2p2: %v", err)
			}
			if val != tt.val {
				t.Errorf("val = %d, want %d", val, tt.val)
			}
			if n != tt.size {
				t.Errorf("size = %d, want %d", n, tt.size)
			}
			// Round-trip: writeVarSizeU2p2 picks narrow when the value
			// fits, wide otherwise; writeVarSizeU2p2Wide always emits
			// the wide form. We pick the writer that matches the input
			// width so the round-trip succeeds.
			isWide := tt.bytes[0]&0x80 != 0
			var out []byte
			if isWide {
				out = writeVarSizeU2p2Wide(tt.val)
			} else {
				out = writeVarSizeU2p2(tt.val)
			}
			if !bytes.Equal(out, tt.bytes) {
				t.Errorf("write round-trip (wide=%v):\n got % x\nwant % x", isWide, out, tt.bytes)
			}
		})
	}
}

func TestVILinkRef_Compact(t *testing.T) {
	v, n, err := decodeVILinkRef([]byte{0x42, 0xff, 0xff})
	if err != nil {
		t.Fatalf("decodeVILinkRef: %v", err)
	}
	if n != 1 {
		t.Errorf("n = %d, want 1", n)
	}
	if v.FlagBt != 0x42 {
		t.Errorf("FlagBt = %#x", v.FlagBt)
	}
	if v.HasExplicit {
		t.Errorf("HasExplicit = true, want false")
	}
	out := encodeVILinkRef(v)
	if !bytes.Equal(out, []byte{0x42}) {
		t.Errorf("encode = % x, want [42]", out)
	}
}

func TestVILinkRef_Explicit(t *testing.T) {
	in := make([]byte, 25)
	in[0] = 0xff
	// Field4Raw = 0x01020304
	in[1], in[2], in[3], in[4] = 1, 2, 3, 4
	v, n, err := decodeVILinkRef(in)
	if err != nil {
		t.Fatalf("decodeVILinkRef: %v", err)
	}
	if n != 25 {
		t.Errorf("n = %d, want 25", n)
	}
	if !v.HasExplicit {
		t.Errorf("HasExplicit = false, want true")
	}
	if v.Field4Raw != 0x01020304 {
		t.Errorf("Field4Raw = %#x", v.Field4Raw)
	}
	out := encodeVILinkRef(v)
	if !bytes.Equal(out, in) {
		t.Errorf("encode mismatch:\n got % x\nwant % x", out, in)
	}
}
