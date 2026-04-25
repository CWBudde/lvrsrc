// Package vits implements the codec for the "VITS" resource — Virtual
// Instrument Tag Strings, a list of (name, LVVariant) pairs LabVIEW uses
// to attach miscellaneous settings and metadata to a VI.
//
// Layout per pylabview LVblock.py:7015-7120:
//
//   - 4-byte big-endian count.
//   - N entries, each:
//   - LStr name: 4-byte big-endian length + that many name bytes.
//   - 4-byte big-endian variant length.
//   - That many variant bytes (opaque to this codec).
//
// LVVariant content is a tagged datafill object whose interpretation
// requires the full pylabview LVdatafill machinery (consolidated type
// list, instance datafill nodes, etc.). Per Phase 6.3i scope, this codec
// preserves the variant bytes verbatim and exposes only the envelope:
// names plus opaque variant payloads. That is enough to enumerate VITS
// keys, surface them in the demo's resource list, and round-trip the
// payload byte-for-byte.
//
// Versions earlier than LV 6.5 inserted four zero bytes between the
// name LStr and the variant payload. The shipped corpus is uniformly
// modern, so this codec does not yet handle that legacy quirk; if such
// fixtures appear, the decoder will need a context-version switch.
//
// Safety tier: 1 (read-only). Mutating variant payloads requires
// understanding their content.
//
// See docs/resources/vits.md for layout, semantics, and open questions.
package vits

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "VITS"

// Value is the decoded form of a VITS payload.
type Value struct {
	// Entries lists the tag-string entries in their original order.
	Entries []TagEntry
}

// TagEntry is one (name, variant) pair from a VITS payload. The variant
// content is preserved as opaque bytes; callers that want to inspect or
// mutate it need a separate LVVariant decoder, which is out of scope for
// Phase 6.3.
type TagEntry struct {
	// Name is the raw bytes of the entry's LStr name. Encoding is the
	// VI's text encoding (see pylabview's `vi.textEncoding`); the codec
	// keeps the bytes opaque and lets callers decode at the right
	// moment.
	Name []byte
	// Variant is the raw bytes of the LVVariant payload that follows
	// the name. Length is determined by the variant's own 4-byte
	// length prefix (which is *not* included in this slice — the codec
	// re-emits the prefix on encode).
	Variant []byte
}

// Codec implements codecs.ResourceCodec for the VITS resource.
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
	if len(payload) < 4 {
		return nil, fmt.Errorf("VITS: payload too short: %d bytes (need at least 4 for count)", len(payload))
	}
	count := binary.BigEndian.Uint32(payload[:4])
	pos := 4

	v := Value{Entries: make([]TagEntry, 0, count)}
	for i := range count {
		if pos+4 > len(payload) {
			return nil, fmt.Errorf("VITS: entry %d name length truncated at offset %d", i, pos)
		}
		nameLen := binary.BigEndian.Uint32(payload[pos : pos+4])
		pos += 4
		if pos+int(nameLen) > len(payload) {
			return nil, fmt.Errorf("VITS: entry %d name truncated: need %d bytes at offset %d, payload size %d", i, nameLen, pos, len(payload))
		}
		name := make([]byte, nameLen)
		copy(name, payload[pos:pos+int(nameLen)])
		pos += int(nameLen)

		if pos+4 > len(payload) {
			return nil, fmt.Errorf("VITS: entry %d variant length truncated at offset %d", i, pos)
		}
		variantLen := binary.BigEndian.Uint32(payload[pos : pos+4])
		pos += 4
		if pos+int(variantLen) > len(payload) {
			return nil, fmt.Errorf("VITS: entry %d variant truncated: need %d bytes at offset %d, payload size %d", i, variantLen, pos, len(payload))
		}
		variant := make([]byte, variantLen)
		copy(variant, payload[pos:pos+int(variantLen)])
		pos += int(variantLen)

		v.Entries = append(v.Entries, TagEntry{Name: name, Variant: variant})
	}
	if pos != len(payload) {
		return nil, fmt.Errorf("VITS: %d trailing byte(s) after %d entr(ies)", len(payload)-pos, count)
	}
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
			return nil, fmt.Errorf("VITS: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("VITS: Encode expected Value or *Value, got %T", value)
	}

	total := 4
	for _, e := range v.Entries {
		total += 4 + len(e.Name) + 4 + len(e.Variant)
	}
	out := make([]byte, total)
	binary.BigEndian.PutUint32(out[:4], uint32(len(v.Entries)))
	pos := 4
	for _, e := range v.Entries {
		binary.BigEndian.PutUint32(out[pos:pos+4], uint32(len(e.Name)))
		pos += 4
		copy(out[pos:], e.Name)
		pos += len(e.Name)
		binary.BigEndian.PutUint32(out[pos:pos+4], uint32(len(e.Variant)))
		pos += 4
		copy(out[pos:], e.Variant)
		pos += len(e.Variant)
	}
	return out, nil
}

// Validate reports structural issues by re-running Decode.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "vits.payload.malformed",
			Message:  fmt.Sprintf("VITS payload could not be parsed: %v", err),
			Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
		}}
	}
	return nil
}
