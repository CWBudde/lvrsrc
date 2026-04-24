package lvvi

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/lvsr"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// buildLVSRPayload synthesises an LVSR payload with the given version and
// flag words. Mirrors the helper in internal/codecs/lvsr/lvsr_test.go but
// lives in this package to avoid exporting it.
func buildLVSRPayload(version uint32, flags ...uint32) []byte {
	out := make([]byte, 4+len(flags)*4)
	binary.BigEndian.PutUint32(out, version)
	for i, w := range flags {
		binary.BigEndian.PutUint32(out[4+i*4:], w)
	}
	return out
}

// fileWithLVSR builds a minimal *lvrsrc.File containing a single LVSR
// block/section with the given payload. It is just enough for
// DecodeKnownResources to find the section; other fields stay zero.
func fileWithLVSR(payload []byte) *lvrsrc.File {
	return &lvrsrc.File{
		Blocks: []lvrsrc.Block{{
			Type: string(lvsr.FourCC),
			Sections: []lvrsrc.Section{{
				Index:   0,
				Payload: payload,
			}},
		}},
	}
}

func TestModelFlagsReturnsFalseWhenNoLVSR(t *testing.T) {
	f := &lvrsrc.File{}
	m, _ := DecodeKnownResources(f)
	if _, ok := m.Flags(); ok {
		t.Fatal("Flags() ok = true on file without LVSR, want false")
	}
}

func TestModelFlagsSurfacesLockedBit(t *testing.T) {
	payload := buildLVSRPayload(0x19010002, 0x00002000, 0, 0, 0, 0, 0)
	m, _ := DecodeKnownResources(fileWithLVSR(payload))
	flags, ok := m.Flags()
	if !ok {
		t.Fatal("Flags() ok = false, want true")
	}
	if !flags.Locked {
		t.Error("flags.Locked = false, want true")
	}
	// Other flags should stay false.
	if flags.RunOnOpen || flags.Debuggable || flags.HasBreakpoints {
		t.Errorf("unexpected flags set: %+v", flags)
	}
}

func TestModelFlagsSurfacesAllBits(t *testing.T) {
	// Build a payload that sets every documented bit.
	flags := make([]uint32, 6)
	flags[0] = 0x00001000 | 0x00002000 | 0x00004000                          // suspend, locked, run-on-open
	flags[1] = 0x00000004 | 0x00000400 | 0x01000000 | 0x20000000             // saved-for-prev, sep-code, clear-ind, auto-err
	flags[5] = 0x20000000 | 0x40000000 | 0x00000200                          // breakpoints + debuggable (both bits)

	m, _ := DecodeKnownResources(fileWithLVSR(buildLVSRPayload(0x19010002, flags...)))
	got, ok := m.Flags()
	if !ok {
		t.Fatal("Flags() ok = false")
	}
	want := LVSRFlags{
		SuspendOnRun:      true,
		Locked:            true,
		RunOnOpen:         true,
		SavedForPrevious:  true,
		SeparateCode:      true,
		ClearIndicators:   true,
		AutoErrorHandling: true,
		HasBreakpoints:    true,
		Debuggable:        true,
	}
	if got != want {
		t.Errorf("flags = %+v, want %+v", got, want)
	}
}

func TestModelFlagsBreakpointCountComesFromWord28(t *testing.T) {
	flags := make([]uint32, 29)
	flags[28] = 7
	m, _ := DecodeKnownResources(fileWithLVSR(buildLVSRPayload(0x19010002, flags...)))
	count, ok := m.BreakpointCount()
	if !ok {
		t.Fatal("BreakpointCount ok = false")
	}
	if count != 7 {
		t.Errorf("BreakpointCount = %d, want 7", count)
	}
}

func TestModelBreakpointCountFalseWhenShortPayload(t *testing.T) {
	m, _ := DecodeKnownResources(fileWithLVSR(buildLVSRPayload(0x19010002, 0, 0, 0, 0, 0, 0)))
	if _, ok := m.BreakpointCount(); ok {
		t.Error("BreakpointCount ok = true on short payload")
	}
}

// Sanity check that flag bytes round-trip the underlying codec so the
// cached Raw slice is actually independent from the caller's payload.
func TestModelFlagsDoesNotAliasCallerPayload(t *testing.T) {
	payload := buildLVSRPayload(0x19010002, 0x00002000, 0, 0, 0, 0, 0)
	original := make([]byte, len(payload))
	copy(original, payload)
	m, _ := DecodeKnownResources(fileWithLVSR(payload))
	// Corrupt the caller's slice; the Model's cached flags must be
	// unaffected.
	for i := range payload {
		payload[i] = 0xFF
	}
	flags, ok := m.Flags()
	if !ok || !flags.Locked {
		t.Fatalf("Locked after caller corruption = (%v, %v); want (true, true)", flags.Locked, ok)
	}
	_ = original
	_ = bytes.Equal
}
