package codecs

import (
	"bytes"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/rsrcwire"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// fakeCodec is a minimal ResourceCodec used in registry tests.
type fakeCodec struct {
	cap         Capability
	decodeValue any
	decodeErr   error
	encodeBytes []byte
	encodeErr   error
	issues      []validate.Issue
}

func (f fakeCodec) Capability() Capability { return f.cap }
func (f fakeCodec) Decode(_ Context, _ []byte) (any, error) {
	return f.decodeValue, f.decodeErr
}

func (f fakeCodec) Encode(_ Context, _ any) ([]byte, error) {
	return f.encodeBytes, f.encodeErr
}

func (f fakeCodec) Validate(_ Context, _ []byte) []validate.Issue {
	return f.issues
}

func TestVersionRangeContains(t *testing.T) {
	cases := []struct {
		name string
		r    VersionRange
		v    uint16
		want bool
	}{
		{"unbounded accepts zero", VersionRange{Min: 0, Max: 0}, 0, true},
		{"unbounded accepts high", VersionRange{Min: 0, Max: 0}, 0xFFFF, true},
		{"bounded in range", VersionRange{Min: 10, Max: 20}, 15, true},
		{"bounded min inclusive", VersionRange{Min: 10, Max: 20}, 10, true},
		{"bounded max inclusive", VersionRange{Min: 10, Max: 20}, 20, true},
		{"below min rejected", VersionRange{Min: 10, Max: 20}, 9, false},
		{"above max rejected", VersionRange{Min: 10, Max: 20}, 21, false},
		{"unbounded upper only", VersionRange{Min: 10, Max: 0}, 0xFFFF, true},
		{"unbounded upper rejects below min", VersionRange{Min: 10, Max: 0}, 5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.Contains(tc.v); got != tc.want {
				t.Fatalf("Contains(%d) = %v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestRegistryLookupFallsBackToOpaque(t *testing.T) {
	r := New()
	c := r.Lookup("LVSR")
	if _, ok := c.(OpaqueCodec); !ok {
		t.Fatalf("Lookup of unregistered FourCC returned %T, want OpaqueCodec", c)
	}
	if r.Has("LVSR") {
		t.Fatalf("Has(unregistered) = true, want false")
	}
}

func TestRegistryRegisterAndLookup(t *testing.T) {
	r := New()

	fc1 := fakeCodec{cap: Capability{FourCC: "BDPW", Safety: SafetyTier1}}
	fc2 := fakeCodec{cap: Capability{FourCC: "LVSR", Safety: SafetyTier2}}
	r.Register(fc1)
	r.Register(fc2)

	if !r.Has("BDPW") || !r.Has("LVSR") {
		t.Fatalf("Has returned false for registered codec")
	}

	got := r.Lookup("BDPW")
	if got.Capability().FourCC != "BDPW" {
		t.Fatalf("Lookup(BDPW).FourCC = %q, want BDPW", got.Capability().FourCC)
	}

	caps := r.Capabilities()
	if len(caps) != 2 {
		t.Fatalf("Capabilities len = %d, want 2", len(caps))
	}
	// Sorted by FourCC: BDPW before LVSR.
	if caps[0].FourCC != "BDPW" || caps[1].FourCC != "LVSR" {
		t.Fatalf("Capabilities order = [%q, %q], want [BDPW, LVSR]", caps[0].FourCC, caps[1].FourCC)
	}
}

func TestRegistryRegisterPanicsOnDuplicate(t *testing.T) {
	r := New()
	r.Register(fakeCodec{cap: Capability{FourCC: "LVSR"}})

	defer func() {
		if recover() == nil {
			t.Fatalf("duplicate Register did not panic")
		}
	}()
	r.Register(fakeCodec{cap: Capability{FourCC: "LVSR"}})
}

func TestRegistryRegisterPanicsOnEmptyFourCC(t *testing.T) {
	r := New()
	defer func() {
		if recover() == nil {
			t.Fatalf("Register with empty FourCC did not panic")
		}
	}()
	r.Register(fakeCodec{cap: Capability{FourCC: ""}})
}

func TestOpaqueCodecCapability(t *testing.T) {
	c := OpaqueCodec{}.Capability()
	if c.FourCC != "" {
		t.Fatalf("OpaqueCodec FourCC = %q, want empty", c.FourCC)
	}
	if c.Safety != SafetyTier1 {
		t.Fatalf("OpaqueCodec Safety = %v, want SafetyTier1", c.Safety)
	}
	if !c.ReadVersions.Contains(0) || !c.ReadVersions.Contains(0xFFFF) {
		t.Fatalf("OpaqueCodec ReadVersions should be unbounded")
	}
}

func TestOpaqueCodecDecodeReturnsDefensiveCopy(t *testing.T) {
	payload := []byte{1, 2, 3, 4}
	ctx := Context{FileVersion: 0x0800, Kind: rsrcwire.FileKindVI}

	v, err := OpaqueCodec{}.Decode(ctx, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	got, ok := v.([]byte)
	if !ok {
		t.Fatalf("Decode returned %T, want []byte", v)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Decode result = %v, want %v", got, payload)
	}
	// Mutate the returned slice; original must remain unchanged.
	got[0] = 0xFF
	if payload[0] != 1 {
		t.Fatalf("mutating Decode result mutated input: payload[0] = %x", payload[0])
	}
}

func TestOpaqueCodecEncodeRoundTrip(t *testing.T) {
	ctx := Context{}
	in := []byte{5, 6, 7}
	out, err := OpaqueCodec{}.Encode(ctx, in)
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	if !bytes.Equal(out, in) {
		t.Fatalf("Encode result = %v, want %v", out, in)
	}
	// Mutate returned slice; input must remain unchanged.
	out[0] = 0xFF
	if in[0] != 5 {
		t.Fatalf("mutating Encode result mutated input: in[0] = %x", in[0])
	}
}

func TestOpaqueCodecEncodeRejectsNonBytes(t *testing.T) {
	_, err := OpaqueCodec{}.Encode(Context{}, "not bytes")
	if err == nil {
		t.Fatalf("Encode of string did not error")
	}
	if err.Error() == "" {
		t.Fatalf("Encode error message unexpectedly empty")
	}
}

func TestOpaqueCodecValidateNeverReportsIssues(t *testing.T) {
	issues := OpaqueCodec{}.Validate(Context{}, []byte{1, 2, 3})
	if len(issues) != 0 {
		t.Fatalf("Validate returned %d issues, want 0", len(issues))
	}
}
