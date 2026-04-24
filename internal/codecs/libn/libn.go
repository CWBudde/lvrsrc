// Package libn implements the codec for the "LIBN" resource — Library
// Names, the list of `.lvlib` files this VI is a member of.
//
// References: pylabview LVblock.py:4683-4756 (LIBN class). Pascal-style
// strings with a 1-byte length prefix and no padding (padto=1 in
// pylabview's readPStr / preparePStr helpers).
//
// See docs/resources/libn.md for layout and semantics.
package libn

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "LIBN"

// Value is the decoded form of a LIBN payload.
type Value struct {
	// Names lists every library name in the order they appear on disk.
	// Each name is the raw byte content of the Pascal string with no
	// charset normalisation; pylabview leaves the encoding to the
	// caller's `vi.textEncoding`, which we expose as []byte for now.
	Names [][]byte
}

// Codec implements codecs.ResourceCodec for the LIBN resource.
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
		return nil, fmt.Errorf("LIBN: payload too short: %d bytes (need at least 4 for count)", len(payload))
	}
	count := binary.BigEndian.Uint32(payload[:4])
	pos := 4

	names := make([][]byte, 0, count)
	for i := range count {
		if pos >= len(payload) {
			return nil, fmt.Errorf("LIBN: payload truncated reading length of name %d", i)
		}
		n := int(payload[pos])
		pos++
		if pos+n > len(payload) {
			return nil, fmt.Errorf("LIBN: payload truncated reading bytes of name %d (need %d, have %d)", i, n, len(payload)-pos)
		}
		entry := make([]byte, n)
		copy(entry, payload[pos:pos+n])
		names = append(names, entry)
		pos += n
	}
	if pos != len(payload) {
		return nil, fmt.Errorf("LIBN: %d trailing byte(s) after %d name(s)", len(payload)-pos, count)
	}
	return Value{Names: names}, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("LIBN: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("LIBN: Encode expected Value or *Value, got %T", value)
	}

	total := 4
	for i, name := range v.Names {
		if len(name) > 0xFF {
			return nil, fmt.Errorf("LIBN: name[%d] is %d bytes, max 255 (Pascal-string limit)", i, len(name))
		}
		total += 1 + len(name)
	}
	out := make([]byte, 0, total)
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, uint32(len(v.Names)))
	out = append(out, count...)
	for _, name := range v.Names {
		out = append(out, byte(len(name)))
		out = append(out, name...)
	}
	return out, nil
}

// Validate reports structural issues by re-running Decode and surfacing
// any error it produces.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "libn.payload.malformed",
			Message:  fmt.Sprintf("LIBN payload could not be parsed: %v", err),
			Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
		}}
	}
	return nil
}
