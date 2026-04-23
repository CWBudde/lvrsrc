// Package icon implements typed codecs for the fixed-size LabVIEW icon
// resources:
//
//   - "ICON": 32x32, 1-bit monochrome
//   - "icl4": 32x32, 4-bit indexed color
//   - "icl8": 32x32, 8-bit indexed color
//
// The payloads are raw raster planes with no per-resource header. This package
// exposes them as normalized per-pixel indices in row-major order.
package icon

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

const (
	MonoFourCC   codecs.FourCC = "ICON"
	Color4FourCC codecs.FourCC = "icl4"
	Color8FourCC codecs.FourCC = "icl8"

	Width      = 32
	Height     = 32
	PixelCount = Width * Height
)

// Value is the decoded form of an icon resource. Pixels are normalized to one
// byte per pixel regardless of the on-disk bit depth.
type Value struct {
	FourCC       codecs.FourCC
	Width        int
	Height       int
	BitsPerPixel int
	Pixels       []byte
}

// MonoCodec implements the "ICON" 1-bit icon resource.
type MonoCodec struct{}

// Color4Codec implements the "icl4" 4-bit icon resource.
type Color4Codec struct{}

// Color8Codec implements the "icl8" 8-bit icon resource.
type Color8Codec struct{}

type spec struct {
	fourCC       codecs.FourCC
	bitsPerPixel int
	rawSize      int
	issueCode    string
}

func (MonoCodec) Capability() codecs.Capability   { return monoSpec.capability() }
func (Color4Codec) Capability() codecs.Capability { return color4Spec.capability() }
func (Color8Codec) Capability() codecs.Capability { return color8Spec.capability() }

func (MonoCodec) Decode(_ codecs.Context, payload []byte) (any, error) {
	return decode(monoSpec, payload)
}
func (Color4Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	return decode(color4Spec, payload)
}
func (Color8Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	return decode(color8Spec, payload)
}

func (MonoCodec) Encode(_ codecs.Context, value any) ([]byte, error) { return encode(monoSpec, value) }
func (Color4Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	return encode(color4Spec, value)
}
func (Color8Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	return encode(color8Spec, value)
}

func (MonoCodec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	return validatePayload(monoSpec, payload)
}
func (Color4Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	return validatePayload(color4Spec, payload)
}
func (Color8Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	return validatePayload(color8Spec, payload)
}

var (
	monoSpec   = spec{fourCC: MonoFourCC, bitsPerPixel: 1, rawSize: 128, issueCode: "icon.payload.size"}
	color4Spec = spec{fourCC: Color4FourCC, bitsPerPixel: 4, rawSize: 512, issueCode: "icl4.payload.size"}
	color8Spec = spec{fourCC: Color8FourCC, bitsPerPixel: 8, rawSize: 1024, issueCode: "icl8.payload.size"}
)

func (s spec) capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        s.fourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier2,
	}
}

func decode(s spec, payload []byte) (Value, error) {
	if len(payload) != s.rawSize {
		return Value{}, fmt.Errorf("%s: payload size = %d, want %d", s.fourCC, len(payload), s.rawSize)
	}

	v := Value{
		FourCC:       s.fourCC,
		Width:        Width,
		Height:       Height,
		BitsPerPixel: s.bitsPerPixel,
		Pixels:       make([]byte, 0, PixelCount),
	}

	switch s.bitsPerPixel {
	case 1:
		for _, b := range payload {
			for shift := 7; shift >= 0; shift-- {
				v.Pixels = append(v.Pixels, (b>>shift)&0x01)
			}
		}
	case 4:
		for _, b := range payload {
			v.Pixels = append(v.Pixels, b>>4, b&0x0F)
		}
	case 8:
		v.Pixels = append(v.Pixels, payload...)
	default:
		return Value{}, fmt.Errorf("%s: unsupported bits-per-pixel %d", s.fourCC, s.bitsPerPixel)
	}

	return v, nil
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
	if v.Width != 0 && v.Width != Width {
		return nil, fmt.Errorf("%s: Value.Width = %d, want %d", s.fourCC, v.Width, Width)
	}
	if v.Height != 0 && v.Height != Height {
		return nil, fmt.Errorf("%s: Value.Height = %d, want %d", s.fourCC, v.Height, Height)
	}
	if v.BitsPerPixel != 0 && v.BitsPerPixel != s.bitsPerPixel {
		return nil, fmt.Errorf("%s: Value.BitsPerPixel = %d, want %d", s.fourCC, v.BitsPerPixel, s.bitsPerPixel)
	}
	if len(v.Pixels) != PixelCount {
		return nil, fmt.Errorf("%s: len(Pixels) = %d, want %d", s.fourCC, len(v.Pixels), PixelCount)
	}

	out := make([]byte, s.rawSize)
	switch s.bitsPerPixel {
	case 1:
		for i, px := range v.Pixels {
			if px > 1 {
				return nil, fmt.Errorf("%s: pixel[%d] = %d, want 0..1", s.fourCC, i, px)
			}
			if px == 1 {
				out[i/8] |= 1 << (7 - (i % 8))
			}
		}
	case 4:
		for i, px := range v.Pixels {
			if px > 0x0F {
				return nil, fmt.Errorf("%s: pixel[%d] = %d, want 0..15", s.fourCC, i, px)
			}
			if i%2 == 0 {
				out[i/2] |= px << 4
			} else {
				out[i/2] |= px
			}
		}
	case 8:
		copy(out, v.Pixels)
	}

	return out, nil
}

func validatePayload(s spec, payload []byte) []validate.Issue {
	if len(payload) == s.rawSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     s.issueCode,
		Message:  fmt.Sprintf("%s payload is %d bytes, want %d", s.fourCC, len(payload), s.rawSize),
		Location: validate.IssueLocation{Area: string(s.fourCC), BlockType: string(s.fourCC)},
	}}
}
