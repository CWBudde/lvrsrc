package lvsr

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

func TestCapability(t *testing.T) {
	c := Codec{}.Capability()
	if c.FourCC != FourCC {
		t.Fatalf("FourCC = %q, want %q", c.FourCC, FourCC)
	}
	if c.Safety != codecs.SafetyTier1 {
		t.Fatalf("Safety = %v, want SafetyTier1 (read-only)", c.Safety)
	}
}

// buildPayload builds an LVSR payload with the given version and flag words.
// flags is expanded to a big-endian uint32 byte run.
func buildPayload(version uint32, flags ...uint32) []byte {
	out := make([]byte, 4+len(flags)*4)
	binary.BigEndian.PutUint32(out, version)
	for i, w := range flags {
		binary.BigEndian.PutUint32(out[4+i*4:], w)
	}
	return out
}

func decodeValue(t *testing.T, payload []byte) Value {
	t.Helper()
	raw, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	v, ok := raw.(Value)
	if !ok {
		t.Fatalf("Decode returned %T, want Value", raw)
	}
	return v
}

func TestDecodeExtractsVersionAndRaw(t *testing.T) {
	payload := buildPayload(0x19010002, 0xAABBCCDD, 0x11223344)
	v := decodeValue(t, payload)
	if v.Version != 0x19010002 {
		t.Errorf("Version = %#08x, want 0x19010002", v.Version)
	}
	wantRaw := payload[4:]
	if !bytes.Equal(v.Raw, wantRaw) {
		t.Errorf("Raw = %x, want %x", v.Raw, wantRaw)
	}
}

func TestDecodeRejectsTooShort(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3} {
		payload := make([]byte, size)
		if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
			t.Errorf("Decode(%d-byte payload) = nil, want error", size)
		}
	}
}

func TestDecodeAcceptsVersionOnlyPayload(t *testing.T) {
	payload := buildPayload(0x19010002)
	v := decodeValue(t, payload)
	if v.Version != 0x19010002 {
		t.Errorf("Version = %#08x, want 0x19010002", v.Version)
	}
	if len(v.Raw) != 0 {
		t.Errorf("len(Raw) = %d, want 0", len(v.Raw))
	}
}

func TestFlagAccessors(t *testing.T) {
	// Each case flips exactly one flag in a zero payload and verifies the
	// matching accessor returns true, plus that all other accessors remain
	// false. Flag map ported from pylavi/resource_types.py:100-108.
	cases := []struct {
		name   string
		word   int
		mask   uint32
		getter func(Value) bool
	}{
		{"SuspendOnRun", 0, 0x00001000, Value.SuspendOnRun},
		{"Locked", 0, 0x00002000, Value.Locked},
		{"RunOnOpen", 0, 0x00004000, Value.RunOnOpen},
		{"SavedForPrevious", 1, 0x00000004, Value.SavedForPrevious},
		{"SeparateCode", 1, 0x00000400, Value.SeparateCode},
		{"ClearIndicators", 1, 0x01000000, Value.ClearIndicators},
		{"AutoErrorHandling", 1, 0x20000000, Value.AutoErrorHandling},
		{"HasBreakpoints", 5, 0x20000000, Value.HasBreakpoints},
		{"DebuggableHiBit", 5, 0x40000000, Value.Debuggable},
		{"DebuggableLoBit", 5, 0x00000200, Value.Debuggable},
	}

	// Payload wide enough to reach word 5.
	const words = 6
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flags := make([]uint32, words)
			flags[tc.word] = tc.mask
			v := decodeValue(t, buildPayload(0, flags...))
			if !tc.getter(v) {
				t.Fatalf("%s() = false, want true", tc.name)
			}

			// Fresh zero payload: every accessor should return false.
			zero := decodeValue(t, buildPayload(0, make([]uint32, words)...))
			for _, other := range cases {
				if other.getter(zero) {
					t.Errorf("%s returned true on zero payload", other.name)
				}
			}
		})
	}
}

func TestDebuggableMatchesPylaviMaskSemantics(t *testing.T) {
	// pylavi treats DEBUGGABLE as mask 0x40000200: "any bit set" means true.
	// So either bit alone should register, both should register, and neither
	// should not.
	const words = 6
	cases := []struct {
		name     string
		word5    uint32
		expected bool
	}{
		{"both bits", 0x40000200, true},
		{"high bit only", 0x40000000, true},
		{"low bit only", 0x00000200, true},
		{"unrelated bit", 0x01000000, false},
		{"zero", 0x00000000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flags := make([]uint32, words)
			flags[5] = tc.word5
			v := decodeValue(t, buildPayload(0, flags...))
			if got := v.Debuggable(); got != tc.expected {
				t.Fatalf("Debuggable() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestBreakpointCountReadsWord28(t *testing.T) {
	// Need 29 flag words to reach word 28.
	flags := make([]uint32, 29)
	flags[28] = 42
	v := decodeValue(t, buildPayload(0, flags...))
	if got, ok := v.BreakpointCount(); !ok || got != 42 {
		t.Fatalf("BreakpointCount = (%d, %v), want (42, true)", got, ok)
	}
}

func TestBreakpointCountAbsentForShortPayload(t *testing.T) {
	// Too-short payload — no word 28. BreakpointCount must report ok=false.
	v := decodeValue(t, buildPayload(0, 0, 0, 0))
	if _, ok := v.BreakpointCount(); ok {
		t.Fatal("BreakpointCount ok = true on short payload, want false")
	}
}

func TestEncodeRoundTripMinimal(t *testing.T) {
	original := buildPayload(0x19010002, 0xAABBCCDD, 0x11223344)
	v := decodeValue(t, original)
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: got %x, want %x", back, original)
	}
}

func TestEncodeRoundTripWide(t *testing.T) {
	// 160-byte payload (corpus size).
	flags := make([]uint32, 39)
	for i := range flags {
		flags[i] = uint32(i * 0x01010101)
	}
	original := buildPayload(0x1B000002, flags...)
	v := decodeValue(t, original)
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original")
	}
}

func TestEncodePreservesOddTrailingByteCount(t *testing.T) {
	// pylabview observed lengths include 68/96/120/136/137 — 137 is not a
	// multiple of 4. Ensure round-trip preserves such an odd tail exactly.
	payload := append(buildPayload(0x1C000000, 0xDEADBEEF),
		0xFE, 0xED, 0xFA, 0xCE, 0x77) // 4 + 4 + 5 = 13 bytes
	v := decodeValue(t, payload)
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, payload) {
		t.Fatalf("Encode != original: got %x, want %x", back, payload)
	}
}

func TestValidateRejectsTooShort(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0x01})
	if len(issues) == 0 {
		t.Fatal("Validate: expected an issue for too-short payload")
	}
	if issues[0].Severity != validate.SeverityError {
		t.Errorf("issue severity = %v, want error", issues[0].Severity)
	}
}

func TestValidateAcceptsWellFormed(t *testing.T) {
	payload := buildPayload(0x19010002, 0, 0, 0, 0, 0, 0)
	if issues := (Codec{}).Validate(codecs.Context{}, payload); len(issues) != 0 {
		t.Fatalf("Validate: got %+v, want no issues", issues)
	}
}
