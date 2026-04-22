// Package codecs defines the resource codec registry that decodes, encodes,
// and validates RSRC block payloads by FourCC type.
//
// Each codec declares its capability (supported file-format versions and
// safety tier) and implements Decode/Encode/Validate. Unknown block types
// fall back to OpaqueCodec, which preserves raw bytes unchanged.
package codecs

import (
	"github.com/CWBudde/lvrsrc/internal/rsrcwire"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is a 4-character block type identifier. It is an alias for string so
// it interoperates transparently with rsrcwire.Block.Type and pkg/lvrsrc types
// without conversions at map-key or struct-field boundaries.
type FourCC = string

// SafetyTier classifies codecs according to docs/safety-model.md.
type SafetyTier int

const (
	// SafetyTier1 codecs are read-only: Decode and Validate are supported,
	// but Encode is not guaranteed to preserve file invariants.
	SafetyTier1 SafetyTier = 1
	// SafetyTier2 codecs support safe metadata edits: Encode preserves
	// codec-level invariants and the surrounding file structure.
	SafetyTier2 SafetyTier = 2
	// SafetyTier3 codecs perform raw/unsafe patches: callers opt in
	// explicitly and accept responsibility for correctness.
	SafetyTier3 SafetyTier = 3
)

// VersionRange describes an inclusive range of supported file-format versions.
// Max == 0 means the upper bound is unbounded/unknown.
type VersionRange struct {
	Min uint16
	Max uint16
}

// Contains reports whether v falls within r. Max == 0 is treated as unbounded.
func (r VersionRange) Contains(v uint16) bool {
	if v < r.Min {
		return false
	}
	if r.Max != 0 && v > r.Max {
		return false
	}
	return true
}

// Capability is the static description of a codec.
type Capability struct {
	FourCC        FourCC
	ReadVersions  VersionRange
	WriteVersions VersionRange
	Safety        SafetyTier
}

// Context is passed to every codec call so codecs can adapt to the file
// version and kind. Phase 4.2 will introduce a richer Version type.
type Context struct {
	FileVersion uint16
	Kind        rsrcwire.FileKind
}

// ResourceCodec decodes, encodes, and validates the payload of a single
// block type. Implementations should be stateless and safe for concurrent use.
type ResourceCodec interface {
	Capability() Capability
	Decode(ctx Context, payload []byte) (any, error)
	Encode(ctx Context, value any) ([]byte, error)
	Validate(ctx Context, payload []byte) []validate.Issue
}
