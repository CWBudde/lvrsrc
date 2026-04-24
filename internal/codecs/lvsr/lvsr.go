// Package lvsr implements the codec for the "LVSR" resource — LabVIEW's
// Save Record, a small header that carries the VI's stored version and a
// run of flag words.
//
// See docs/resources/lvsr.md for the byte layout, flag-bit table, and
// references. The flag-bit semantics are ported from pylavi's TypeLVSR
// (references/pylavi/pylavi/resource_types.py:96-198) which provides the
// most concise published description. pylabview's LVSRData class
// (references/pylabview/pylabview/LVblock.py:3503+) confirms the first
// flag word is `execFlags` with LibProtected = 1<<13 = Locked and
// RunOnOpen = 1<<14 — matching pylavi's map.
//
// This codec is intentionally narrow: the first four bytes are read as a
// big-endian Version word, every following byte is preserved as an opaque
// Raw slice, and flag accessors project typed booleans out of Raw on
// demand. Encoding is a plain serialize — the codec does not reinterpret
// the tail, so it round-trips byte-for-byte regardless of LabVIEW version
// or unusual trailing sizes.
package lvsr

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "LVSR"

// versionSize is the byte count of the big-endian version word that sits
// at the start of every LVSR payload observed in the corpus.
const versionSize = 4

// Value is the decoded form of an LVSR payload.
type Value struct {
	// Version is the raw big-endian uint32 at offset 0. It carries the
	// LabVIEW version in the same packed BCD-ish layout used elsewhere in
	// the format (see pkg/lvvi.Version). The codec preserves it verbatim
	// on encode; interpretation is the caller's responsibility.
	Version uint32

	// Raw is the remaining payload bytes after the Version word. Flag
	// accessors index this slice as big-endian uint32 words. Length is
	// usually — but not required to be — a multiple of 4; the codec
	// round-trips odd sizes exactly.
	Raw []byte
}

// Codec implements codecs.ResourceCodec for the LVSR resource.
type Codec struct{}

// Capability reports the codec's static metadata.
func (Codec) Capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        FourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier1,
	}
}

// Decode parses payload into a Value.
func (Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	if len(payload) < versionSize {
		return nil, fmt.Errorf("LVSR: payload too short: %d bytes (need at least %d)", len(payload), versionSize)
	}
	raw := make([]byte, len(payload)-versionSize)
	copy(raw, payload[versionSize:])
	return Value{
		Version: binary.BigEndian.Uint32(payload[:versionSize]),
		Raw:     raw,
	}, nil
}

// Encode serializes a Value (by value or pointer) into the wire format.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("LVSR: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("LVSR: Encode expected Value or *Value, got %T", value)
	}

	out := make([]byte, versionSize+len(v.Raw))
	binary.BigEndian.PutUint32(out[:versionSize], v.Version)
	copy(out[versionSize:], v.Raw)
	return out, nil
}

// Validate reports structural issues. The only hard invariant observed in
// the corpus is a minimum payload size of four bytes (version word).
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) >= versionSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "lvsr.payload.too_short",
		Message:  fmt.Sprintf("LVSR payload is %d bytes, need at least %d", len(payload), versionSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}

// --- Flag accessors ---
//
// The flag coordinates are ported verbatim from pylavi's TypeLVSR: a pair
// (word-index, mask) interpreted as "flag is set when any bit of the mask
// is set in the big-endian uint32 at Raw[word*4 : word*4+4]". Accessors
// return false when the raw payload is too short to carry the requested
// word — matching pylavi's get_flag_set None-guard.

// SuspendOnRun reports whether the "suspend when called" flag is set
// (word 0, mask 0x00001000). pylabview calls this bit `HasSetBP` in
// VI_EXEC_FLAGS; pylavi exposes it via suspend_on_run().
func (v Value) SuspendOnRun() bool { return v.flagBit(0, 0x00001000) }

// Locked reports whether the library containing this VI is locked
// (word 0, mask 0x00002000). pylabview calls the same bit
// VI_EXEC_FLAGS.LibProtected.
func (v Value) Locked() bool { return v.flagBit(0, 0x00002000) }

// RunOnOpen reports whether the VI runs automatically when loaded
// (word 0, mask 0x00004000; pylabview VI_EXEC_FLAGS.RunOnOpen).
func (v Value) RunOnOpen() bool { return v.flagBit(0, 0x00004000) }

// SavedForPrevious reports whether the VI was saved for a previous
// LabVIEW version (word 1, mask 0x00000004).
func (v Value) SavedForPrevious() bool { return v.flagBit(1, 0x00000004) }

// SeparateCode reports whether the VI is stored with compiled code
// separate from the source (word 1, mask 0x00000400).
func (v Value) SeparateCode() bool { return v.flagBit(1, 0x00000400) }

// ClearIndicators reports whether unwired indicators are cleared on each
// run (word 1, mask 0x01000000).
func (v Value) ClearIndicators() bool { return v.flagBit(1, 0x01000000) }

// AutoErrorHandling reports whether automatic error handling is enabled
// (word 1, mask 0x20000000).
func (v Value) AutoErrorHandling() bool { return v.flagBit(1, 0x20000000) }

// HasBreakpoints reports whether the VI carries any stored breakpoints
// (word 5, mask 0x20000000).
func (v Value) HasBreakpoints() bool { return v.flagBit(5, 0x20000000) }

// Debuggable reports whether the VI was saved with debugging enabled
// (word 5, mask 0x40000200). pylavi uses a compound mask because either
// bit signals debuggable across LabVIEW versions; this accessor follows
// the same "any bit set" semantics.
func (v Value) Debuggable() bool { return v.flagBit(5, 0x40000200) }

// BreakpointCount returns the integer stored at flag-word index 28, per
// pylavi's BREAKPOINT_COUNT_INDEX. Returns ok=false if the payload is
// too short to reach word 28.
func (v Value) BreakpointCount() (uint32, bool) {
	return v.flagWord(28)
}

// flagWord returns the big-endian uint32 at flag-word index idx in Raw,
// or ok=false when Raw is too short to reach that word.
func (v Value) flagWord(idx int) (uint32, bool) {
	base := idx * 4
	if base+4 > len(v.Raw) {
		return 0, false
	}
	return binary.BigEndian.Uint32(v.Raw[base : base+4]), true
}

// flagBit returns true when any bit of mask is set in the word at idx.
// Returns false when the payload is too short to reach that word.
func (v Value) flagBit(idx int, mask uint32) bool {
	w, ok := v.flagWord(idx)
	if !ok {
		return false
	}
	return w&mask != 0
}
