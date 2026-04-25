// Package livi implements a narrow codec for the "LIvi" resource — the
// VI-level link-info block. It records dependencies between this VI/CTL
// and other VIs, classes, and libraries.
//
// Scope (Phase 7.2): the codec exposes only the stable outer envelope
// (version, file-kind marker, entry count, footer) and preserves the
// inner entry stream as opaque bytes. Per-entry shape is similar to but
// not identical to LIfp/LIbd's, and the corpus shows entry layouts that
// the libd-style boundary heuristic cannot disambiguate without more
// reverse-engineering work. Round-trip preservation is byte-for-byte
// regardless.
//
// The leading 4-byte marker mirrors the file's content kind:
//   - LVIN for `.vi`
//   - LVCC for `.ctl` (control)
//   - LVIT for `.vit` (template) — not in the shipped corpus
//   - LLBV for `.llb` library — not in the shipped corpus
//
// References: pylabview LVblock.py:2426 (LIvi class) + LinkObjRefs base
// at LVblock.py:2248. Per-entry decoding is intentionally deferred to
// Phase 7.3 / Phase 9 where the LinkObjRef family is fully ported.
package livi

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

const (
	// FourCC is the resource type this codec handles.
	FourCC codecs.FourCC = "LIvi"

	headerSize = 10 // u16 version + 4-byte marker + u32 entry count
	footerSize = 2  // u16 trailing value
)

// knownMarkers enumerates the file-kind FourCCs LIvi is observed to
// carry. Decode accepts any 4-byte marker (Encode preserves whatever the
// caller supplies); Validate flags unknown markers as a warning so the
// corpus surface stays documented.
var knownMarkers = map[string]struct{}{
	"LVIN": {}, // VI
	"LVCC": {}, // CTL (control)
	"LVIT": {}, // VIT (template)
	"LLBV": {}, // LLB (library)
}

func isKnownMarker(s string) bool {
	_, ok := knownMarkers[s]
	return ok
}

// Value is the decoded form of an LIvi payload.
type Value struct {
	// Version is the u16 BE version word at offset 0.
	Version uint16
	// Marker is the 4-byte file-kind FourCC at offset 2.
	Marker string
	// EntryCount is the u32 BE count at offset 6. Per-entry parsing is
	// deferred; this is exposed so callers can summarise dependency
	// counts without decoding entries.
	EntryCount uint32
	// Body is the opaque bytes between the header and footer, holding
	// EntryCount entries in their original on-disk form.
	Body []byte
	// Footer is the u16 BE value at the tail of the payload.
	Footer uint16
}

// Codec implements codecs.ResourceCodec for the LIvi resource.
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
	if len(payload) < headerSize+footerSize {
		return nil, fmt.Errorf("LIvi: payload too short: %d bytes (need at least %d)", len(payload), headerSize+footerSize)
	}
	v := Value{
		Version:    binary.BigEndian.Uint16(payload[0:2]),
		Marker:     string(payload[2:6]),
		EntryCount: binary.BigEndian.Uint32(payload[6:10]),
		Footer:     binary.BigEndian.Uint16(payload[len(payload)-footerSize:]),
	}
	bodyLen := len(payload) - headerSize - footerSize
	v.Body = make([]byte, bodyLen)
	copy(v.Body, payload[headerSize:headerSize+bodyLen])
	return v, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("LIvi: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("LIvi: Encode expected Value or *Value, got %T", value)
	}
	if len(v.Marker) != 4 {
		return nil, fmt.Errorf("LIvi: Marker length = %d, want 4", len(v.Marker))
	}
	out := make([]byte, headerSize+len(v.Body)+footerSize)
	binary.BigEndian.PutUint16(out[0:2], v.Version)
	copy(out[2:6], v.Marker)
	binary.BigEndian.PutUint32(out[6:10], v.EntryCount)
	copy(out[headerSize:], v.Body)
	binary.BigEndian.PutUint16(out[len(out)-footerSize:], v.Footer)
	return out, nil
}

// Validate reports structural issues with payload.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)}
	if len(payload) < headerSize+footerSize {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "livi.payload.short",
			Message:  fmt.Sprintf("LIvi payload is %d bytes, need at least %d", len(payload), headerSize+footerSize),
			Location: loc,
		}}
	}
	marker := string(payload[2:6])
	if !isKnownMarker(marker) {
		return []validate.Issue{{
			Severity: validate.SeverityWarning,
			Code:     "livi.marker.unknown",
			Message:  fmt.Sprintf("LIvi marker = %q is not in the known set (LVIN/LVCC/LVIT/LLBV)", marker),
			Location: loc,
		}}
	}
	return nil
}
