// Package conpane implements typed codecs for the LabVIEW connector-pane
// resources:
//
//   - "CONP": 2-byte big-endian connector pane selector/pointer
//   - "CPC2": 2-byte big-endian connector pane count/variant field
//
// The corpus currently justifies treating both as fixed-width uint16 payloads.
package conpane

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

const (
	PointerFourCC codecs.FourCC = "CONP"
	CountFourCC   codecs.FourCC = "CPC2"
	payloadSize                 = 2
)

// Value is the decoded form of a connector-pane scalar resource.
type Value struct {
	FourCC codecs.FourCC
	Value  uint16
}

// PointerCodec implements the "CONP" resource.
type PointerCodec struct{}

// CountCodec implements the "CPC2" resource.
type CountCodec struct{}

type spec struct {
	fourCC    codecs.FourCC
	issueCode string
}

var (
	pointerSpec = spec{fourCC: PointerFourCC, issueCode: "conp.payload.size"}
	countSpec   = spec{fourCC: CountFourCC, issueCode: "cpc2.payload.size"}
)

func (PointerCodec) Capability() codecs.Capability { return pointerSpec.capability() }
func (CountCodec) Capability() codecs.Capability   { return countSpec.capability() }

func (PointerCodec) Decode(_ codecs.Context, payload []byte) (any, error) {
	return decode(pointerSpec, payload)
}

func (CountCodec) Decode(_ codecs.Context, payload []byte) (any, error) {
	return decode(countSpec, payload)
}

func (PointerCodec) Encode(_ codecs.Context, value any) ([]byte, error) {
	return encode(pointerSpec, value)
}

func (CountCodec) Encode(_ codecs.Context, value any) ([]byte, error) {
	return encode(countSpec, value)
}

func (PointerCodec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	return validatePayload(pointerSpec, payload)
}

func (CountCodec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	return validatePayload(countSpec, payload)
}

func (s spec) capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        s.fourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier2,
	}
}

func decode(s spec, payload []byte) (Value, error) {
	if len(payload) != payloadSize {
		return Value{}, fmt.Errorf("%s: payload size = %d, want %d", s.fourCC, len(payload), payloadSize)
	}
	return Value{
		FourCC: s.fourCC,
		Value:  binary.BigEndian.Uint16(payload),
	}, nil
}

func encode(s spec, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("%s: Encode received nil *Value", s.fourCC)
		}
		v = *tv
	default:
		return nil, fmt.Errorf("%s: Encode expected Value or *Value, got %T", s.fourCC, value)
	}

	if v.FourCC != "" && v.FourCC != s.fourCC {
		return nil, fmt.Errorf("%s: Value.FourCC = %q, want %q", s.fourCC, v.FourCC, s.fourCC)
	}

	out := make([]byte, payloadSize)
	binary.BigEndian.PutUint16(out, v.Value)
	return out, nil
}

func validatePayload(s spec, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     s.issueCode,
		Message:  fmt.Sprintf("%s payload is %d bytes, want %d", s.fourCC, len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(s.fourCC), BlockType: string(s.fourCC)},
	}}
}
