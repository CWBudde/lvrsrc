// Package strg implements the codec for the "STRG" resource — the
// LabVIEW VI description string (VI Properties → Documentation → VI
// Description).
//
// See docs/resources/strg.md for the wire layout, references
// (pylabview's StringListBlock / STRG class), and open questions.
//
// This implementation covers the modern LabVIEW ≥ 4.0 single-string layout
// (4-byte BE size + N raw text bytes). Legacy pre-4.0 count-prefixed layout
// is documented but not yet supported.
package strg

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "STRG"

// headerSize is the number of bytes for the uint32 BE size prefix.
const headerSize = 4

// Value is the decoded form of a STRG payload.
type Value struct {
	// Text is the raw description bytes as a Go string. It may contain
	// CR/LF/CRLF line endings; callers that need to display it should
	// normalize as appropriate.
	Text string
}

// Codec implements codecs.ResourceCodec for the STRG resource.
type Codec struct{}

// Capability reports the codec's static metadata.
func (Codec) Capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        FourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier2,
	}
}

// Decode parses payload into a Value. The returned Text is a string copy of
// the payload bytes, safe to retain independently of the input slice.
func (Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	if len(payload) < headerSize {
		return nil, fmt.Errorf("STRG: payload too short: %d bytes (need at least %d)", len(payload), headerSize)
	}
	size := binary.BigEndian.Uint32(payload[:headerSize])
	end := uint64(headerSize) + uint64(size)
	if end > uint64(len(payload)) {
		return nil, fmt.Errorf("STRG: size %d overruns payload (size %d)", size, len(payload))
	}
	return Value{Text: string(payload[headerSize:end])}, nil
}

// Encode serializes a Value (by value or pointer) into the wire format.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("STRG: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("STRG: Encode expected Value or *Value, got %T", value)
	}
	if uint64(len(v.Text)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("STRG: Text length %d exceeds uint32 max", len(v.Text))
	}
	out := make([]byte, headerSize+len(v.Text))
	binary.BigEndian.PutUint32(out[:headerSize], uint32(len(v.Text)))
	copy(out[headerSize:], v.Text)
	return out, nil
}

// Validate reports issues with payload under the rules in
// docs/resources/strg.md.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: "STRG", BlockType: string(FourCC)}
	var issues []validate.Issue

	if len(payload) < headerSize {
		return append(issues, validate.Issue{
			Severity: validate.SeverityError,
			Code:     "strg.payload.short",
			Message:  fmt.Sprintf("STRG payload is %d bytes, need at least %d", len(payload), headerSize),
			Location: loc,
		})
	}

	size := binary.BigEndian.Uint32(payload[:headerSize])
	end := uint64(headerSize) + uint64(size)
	if end > uint64(len(payload)) {
		return append(issues, validate.Issue{
			Severity: validate.SeverityError,
			Code:     "strg.size.overruns_payload",
			Message:  fmt.Sprintf("STRG size %d overruns payload (size %d)", size, len(payload)),
			Location: loc,
		})
	}
	if end < uint64(len(payload)) {
		issues = append(issues, validate.Issue{
			Severity: validate.SeverityWarning,
			Code:     "strg.size.trailing_bytes",
			Message:  fmt.Sprintf("STRG has %d trailing bytes after declared text", uint64(len(payload))-end),
			Location: loc,
		})
	}
	text := payload[headerSize:end]
	for _, b := range text {
		if b < 0x20 && b != '\r' && b != '\n' && b != '\t' {
			issues = append(issues, validate.Issue{
				Severity: validate.SeverityWarning,
				Code:     "strg.text.control_chars",
				Message:  fmt.Sprintf("STRG text contains control byte 0x%02x", b),
				Location: loc,
			})
			break
		}
	}
	return issues
}
