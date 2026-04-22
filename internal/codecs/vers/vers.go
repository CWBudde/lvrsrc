// Package vers implements the codec for the "vers" resource — the LabVIEW
// version stamp that records the major/minor/patch of the LabVIEW
// application that last saved the file, alongside an ASCII version label.
//
// See docs/resources/vers.md for the wire layout, corpus evidence, and open
// questions.
package vers

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "vers"

// stageRelease is the only Stage byte observed across the corpus (0x80).
const stageRelease uint8 = 0x80

// minPayloadSize is the smallest legal payload: 6-byte fixed header +
// 1-byte text length + 0-byte text + 1-byte trailer.
const minPayloadSize = 8

// Value is the decoded form of a vers payload.
type Value struct {
	// Major is the LabVIEW major version as a BCD byte (e.g. 0x25 = 25).
	Major uint8
	// Minor is the minor version number (0-15).
	Minor uint8
	// Patch is the patch number (0-15).
	Patch uint8
	// Stage is the release-stage byte (0x80 = release in all corpus samples).
	Stage uint8
	// Build is the build counter (matches Patch in every corpus sample).
	Build uint8
	// Reserved holds the 16-bit reserved field at offset 4. Observed as 0
	// everywhere; preserved on round-trip.
	Reserved uint16
	// Text is the ASCII version label (e.g. "25.1.2" or "25.0").
	Text string
}

// Codec implements codecs.ResourceCodec for the vers resource.
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

// Decode parses payload into a Value. The returned value's Text is a copy of
// the payload bytes, so the caller may retain it freely.
func (Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	if len(payload) < minPayloadSize {
		return nil, fmt.Errorf("vers: payload too short: %d bytes (need at least %d)", len(payload), minPayloadSize)
	}
	v := Value{
		Major:    payload[0],
		Minor:    payload[1] >> 4,
		Patch:    payload[1] & 0x0F,
		Stage:    payload[2],
		Build:    payload[3],
		Reserved: binary.BigEndian.Uint16(payload[4:6]),
	}
	textLen := int(payload[6])
	textEnd := 7 + textLen
	if textEnd+1 > len(payload) {
		return nil, fmt.Errorf("vers: text length %d overruns payload (size %d)", textLen, len(payload))
	}
	v.Text = string(payload[7:textEnd])
	return v, nil
}

// Encode serializes a Value (passed by value or pointer) into the wire format.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("vers: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("vers: Encode expected Value or *Value, got %T", value)
	}

	if v.Minor > 0x0F {
		return nil, fmt.Errorf("vers: Minor %d does not fit in a nibble", v.Minor)
	}
	if v.Patch > 0x0F {
		return nil, fmt.Errorf("vers: Patch %d does not fit in a nibble", v.Patch)
	}
	if len(v.Text) > 0xFF {
		return nil, fmt.Errorf("vers: Text length %d exceeds Pascal-string max 255", len(v.Text))
	}

	out := make([]byte, 0, minPayloadSize+len(v.Text))
	out = append(out,
		v.Major,
		(v.Minor<<4)|(v.Patch&0x0F),
		v.Stage,
		v.Build,
	)
	var reserved [2]byte
	binary.BigEndian.PutUint16(reserved[:], v.Reserved)
	out = append(out, reserved[:]...)
	out = append(out, byte(len(v.Text)))
	out = append(out, v.Text...)
	out = append(out, 0x00) // trailer
	return out, nil
}

// Validate reports issues with payload under the rules documented in
// docs/resources/vers.md.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: "vers", BlockType: string(FourCC)}
	var issues []validate.Issue

	if len(payload) < minPayloadSize {
		return append(issues, validate.Issue{
			Severity: validate.SeverityError,
			Code:     "vers.payload.short",
			Message:  fmt.Sprintf("vers payload is %d bytes, need at least %d", len(payload), minPayloadSize),
			Location: loc,
		})
	}

	textLen := int(payload[6])
	textEnd := 7 + textLen
	if textEnd+1 > len(payload) {
		issues = append(issues, validate.Issue{
			Severity: validate.SeverityError,
			Code:     "vers.text.overruns_payload",
			Message:  fmt.Sprintf("vers text length %d would read past payload (size %d)", textLen, len(payload)),
			Location: loc,
		})
		return issues
	}
	if payload[textEnd] != 0 {
		issues = append(issues, validate.Issue{
			Severity: validate.SeverityError,
			Code:     "vers.trailer.missing",
			Message:  fmt.Sprintf("vers trailer byte is 0x%02x, want 0x00", payload[textEnd]),
			Location: loc,
		})
	}

	if reserved := binary.BigEndian.Uint16(payload[4:6]); reserved != 0 {
		issues = append(issues, validate.Issue{
			Severity: validate.SeverityWarning,
			Code:     "vers.reserved.nonzero",
			Message:  fmt.Sprintf("vers reserved field = 0x%04x, observed 0x0000 in all known samples", reserved),
			Location: loc,
		})
	}
	if stage := payload[2]; stage != stageRelease {
		issues = append(issues, validate.Issue{
			Severity: validate.SeverityWarning,
			Code:     "vers.stage.unknown",
			Message:  fmt.Sprintf("vers stage = 0x%02x, observed 0x80 (release) in all known samples", stage),
			Location: loc,
		})
	}

	text := payload[7:textEnd]
	for _, b := range text {
		if b < 0x20 || b > 0x7E {
			issues = append(issues, validate.Issue{
				Severity: validate.SeverityWarning,
				Code:     "vers.text.nonascii",
				Message:  fmt.Sprintf("vers text contains non-printable byte 0x%02x", b),
				Location: loc,
			})
			break
		}
	}

	// Check text vs decoded major/minor/patch for consistency. Only flag if the
	// text looks like a version and disagrees with the numeric fields.
	major := payload[0]
	minor := payload[1] >> 4
	patch := payload[1] & 0x0F
	expected := formatExpectedText(major, minor, patch)
	if len(text) > 0 && string(text) != expected {
		issues = append(issues, validate.Issue{
			Severity: validate.SeverityWarning,
			Code:     "vers.text.inconsistent",
			Message:  fmt.Sprintf("vers text %q does not match decoded version %q", string(text), expected),
			Location: loc,
		})
	}

	return issues
}

// formatExpectedText renders major.minor[.patch] the way corpus samples do:
// omit the patch segment when patch == 0.
func formatExpectedText(major, minor, patch uint8) string {
	majorHi := major >> 4
	majorLo := major & 0x0F
	if patch == 0 {
		return fmt.Sprintf("%d%d.%d", majorHi, majorLo, minor)
	}
	return fmt.Sprintf("%d%d.%d.%d", majorHi, majorLo, minor, patch)
}
