// Package fphb implements the codec for the "FPHb" resource — the
// Front-Panel Heap. The codec stitches together two layers shipped in
// `internal/codecs/heap`:
//
//   - the ZLIB envelope (`heap.DecodeEnvelope` / `EncodeEnvelope`),
//   - the tag-stream walker (`heap.Walk`) that turns the inflated
//     content into a tree of `heap.Node` records.
//
// Phase 9.4 scope is Tier 1 read-only: the public Value carries both
// the raw envelope (with its Compressed-bytes cache) and the decoded
// tree so the demo and other callers can introspect the heap without
// re-walking on every access. Encode re-emits the original on-disk
// payload byte-for-byte through the Compressed cache.
//
// Per-tag typed payload editing (mutating Tree.Flat[i].Content and
// rebuilding the byte stream) requires a write-side walker that pylabview
// has but we have not yet ported; it lands when Phase 9 expands toward
// Tier 2 mutations. Until then, callers may read the tree freely but
// must not edit it expecting Encode to re-serialise the changes.
//
// References: pylabview LVblock.py:5094-5179 (HeapVerb base class) and
// LVblock.py:5350-5362 (FPHb / BDHb subclasses).
package fphb

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "FPHb"

// Value is the decoded form of an FPHb payload.
type Value struct {
	// Envelope is the ZLIB envelope view (size headers + compressed
	// bytes cache + inflated content).
	Envelope heap.Envelope
	// Tree is the walker's output: every entry in the tag stream
	// projected onto a flat list and a parent/child tree.
	Tree heap.WalkResult
}

// Codec implements codecs.ResourceCodec for the FPHb resource.
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
	env, err := heap.DecodeEnvelope(payload)
	if err != nil {
		return nil, fmt.Errorf("FPHb: %w", err)
	}
	tree, err := heap.Walk(env.Content)
	if err != nil {
		return nil, fmt.Errorf("FPHb: walk tag stream: %w", err)
	}
	return Value{Envelope: env, Tree: tree}, nil
}

// Encode serializes a Value (by value or pointer) back to the on-disk
// byte form. Because Phase 9.4 is read-only, Encode does not rebuild
// the tag stream from Tree; it forwards to heap.EncodeEnvelope which
// reuses Envelope.Compressed when present (preserving the original
// payload byte-for-byte) and recompresses the inflated buffer
// otherwise.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("FPHb: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("FPHb: Encode expected Value or *Value, got %T", value)
	}
	return heap.EncodeEnvelope(v.Envelope)
}

// Validate reports structural issues with payload. Two layers of
// checks: the envelope must decode cleanly, and the tag stream must
// walk without truncation or trailing bytes.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)}
	env, err := heap.DecodeEnvelope(payload)
	if err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "fphb.envelope.invalid",
			Message:  fmt.Sprintf("FPHb envelope could not be parsed: %v", err),
			Location: loc,
		}}
	}
	if _, err := heap.Walk(env.Content); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "fphb.tagstream.invalid",
			Message:  fmt.Sprintf("FPHb tag stream could not be walked: %v", err),
			Location: loc,
		}}
	}
	return nil
}
