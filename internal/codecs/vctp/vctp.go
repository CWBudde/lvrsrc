// Package vctp implements a narrow codec for the "VCTP" resource, which
// stores the type descriptor pool as a declared uncompressed size followed by
// a zlib-compressed payload blob.
//
// This codec intentionally stops at the compressed-pool boundary. It exposes
// the inflated bytes as a typed descriptor blob without claiming to understand
// individual type records yet.
package vctp

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

const (
	// FourCC is the resource type this codec handles.
	FourCC codecs.FourCC = "VCTP"

	headerSize = 4
)

// Value is the decoded form of a VCTP payload.
type Value struct {
	DeclaredSize uint32
	Inflated     []byte
	Compressed   []byte
}

// Codec implements codecs.ResourceCodec for VCTP.
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
	return decodeValue(payload)
}

// Encode serializes a Value (by value or pointer) into the observed wire
// format. If Compressed matches Inflated it is reused verbatim; otherwise the
// inflated bytes are recompressed with zlib.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	v, err := coerceValue(value)
	if err != nil {
		return nil, err
	}
	if v.DeclaredSize != uint32(len(v.Inflated)) {
		return nil, fmt.Errorf("VCTP: DeclaredSize = %d, want inflated length %d", v.DeclaredSize, len(v.Inflated))
	}

	compressed := v.Compressed
	if len(compressed) != 0 {
		inflated, err := inflate(compressed)
		if err != nil {
			return nil, fmt.Errorf("VCTP: stored compressed bytes invalid: %w", err)
		}
		if !bytes.Equal(inflated, v.Inflated) {
			compressed = nil
		}
	}
	if len(compressed) == 0 {
		compressed, err = deflate(v.Inflated)
		if err != nil {
			return nil, err
		}
	}

	out := make([]byte, headerSize+len(compressed))
	binary.BigEndian.PutUint32(out[:headerSize], v.DeclaredSize)
	copy(out[headerSize:], compressed)
	return out, nil
}

// Validate reports structural issues with payload.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: "VCTP", BlockType: string(FourCC)}
	if len(payload) < headerSize {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "vctp.payload.short",
			Message:  fmt.Sprintf("VCTP payload is %d bytes, need at least %d", len(payload), headerSize),
			Location: loc,
		}}
	}

	declaredSize := binary.BigEndian.Uint32(payload[:headerSize])
	inflated, err := inflate(payload[headerSize:])
	if err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "vctp.zlib.invalid",
			Message:  fmt.Sprintf("VCTP compressed payload is not valid zlib data: %v", err),
			Location: loc,
		}}
	}
	if uint32(len(inflated)) != declaredSize {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "vctp.declared_size.mismatch",
			Message:  fmt.Sprintf("VCTP declared size %d does not match inflated size %d", declaredSize, len(inflated)),
			Location: loc,
		}}
	}
	return nil
}

func coerceValue(value any) (Value, error) {
	switch tv := value.(type) {
	case Value:
		return tv, nil
	case *Value:
		if tv == nil {
			return Value{}, fmt.Errorf("VCTP: Encode received nil *Value")
		}
		return *tv, nil
	default:
		return Value{}, fmt.Errorf("VCTP: Encode expected Value or *Value, got %T", value)
	}
}

func decodeValue(payload []byte) (Value, error) {
	if len(payload) < headerSize {
		return Value{}, fmt.Errorf("VCTP: payload too short: %d bytes (need at least %d)", len(payload), headerSize)
	}
	declaredSize := binary.BigEndian.Uint32(payload[:headerSize])
	compressed := append([]byte(nil), payload[headerSize:]...)
	inflated, err := inflate(compressed)
	if err != nil {
		return Value{}, fmt.Errorf("VCTP: inflate compressed payload: %w", err)
	}
	if uint32(len(inflated)) != declaredSize {
		return Value{}, fmt.Errorf("VCTP: declared size %d does not match inflated size %d", declaredSize, len(inflated))
	}
	return Value{
		DeclaredSize: declaredSize,
		Inflated:     inflated,
		Compressed:   compressed,
	}, nil
}

func inflate(compressed []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	inflated, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return inflated, nil
}

func deflate(inflated []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(inflated); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("VCTP: deflate payload: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("VCTP: finalize deflate payload: %w", err)
	}
	return buf.Bytes(), nil
}
