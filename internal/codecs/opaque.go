package codecs

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/validate"
)

// OpaqueCodec is the fallback codec for unknown block types. It preserves raw
// payload bytes unchanged. Its Capability.FourCC is empty because it is
// returned by Registry.Lookup when no codec is registered, not registered itself.
type OpaqueCodec struct{}

// Capability returns a read-only, all-versions capability with an empty FourCC.
func (OpaqueCodec) Capability() Capability {
	return Capability{
		FourCC:        "",
		ReadVersions:  VersionRange{Min: 0, Max: 0},
		WriteVersions: VersionRange{Min: 0, Max: 0},
		Safety:        SafetyTier1,
	}
}

// Decode returns a defensive copy of payload so callers cannot mutate the
// underlying file's backing store.
func (OpaqueCodec) Decode(_ Context, payload []byte) (any, error) {
	out := make([]byte, len(payload))
	copy(out, payload)
	return out, nil
}

// Encode accepts a []byte value and returns a defensive copy. Any other
// concrete type is a programmer error and returns an error.
func (OpaqueCodec) Encode(_ Context, value any) ([]byte, error) {
	b, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("opaque codec: expected []byte, got %T", value)
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

// Validate reports no issues: opaque payloads have no codec-level invariants.
func (OpaqueCodec) Validate(_ Context, _ []byte) []validate.Issue {
	return nil
}
