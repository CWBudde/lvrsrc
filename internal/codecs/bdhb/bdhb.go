// Package bdhb implements the codec for the "BDHb" resource — the
// Block-Diagram Heap. Like its front-panel sibling FPHb, BDHb wraps a
// ZLIB envelope around a tag-stream tree, and pylabview shares the
// HeapVerb base class for both (LVblock.py:5350-5362 — `BDHb` and
// `FPHb` are sibling subclasses with no parsing differences at the
// envelope or walker layer).
//
// Phase 10.1 ships BDHb at the same Tier 1 read-only level as FPHb:
// every corpus BDHb section round-trips byte-for-byte through the
// shared `internal/codecs/heap` framework, but the codec does not
// rebuild the tag stream from a mutated tree. Per-tag block-diagram
// semantics (primitives, wires, structures) ride on top of the same
// `tags_gen.go` enums; expanding the typed-payload coverage is a
// follow-up rather than a 10.1 deliverable.
package bdhb

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "BDHb"

// Value is the decoded form of a BDHb payload. It mirrors fphb.Value
// since the two resources share the heap framework verbatim.
type Value struct {
	Envelope heap.Envelope
	Tree     heap.WalkResult
}

// Codec implements codecs.ResourceCodec for the BDHb resource.
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
		return nil, fmt.Errorf("BDHb: %w", err)
	}
	tree, err := heap.Walk(env.Content)
	if err != nil {
		return nil, fmt.Errorf("BDHb: walk tag stream: %w", err)
	}
	return Value{Envelope: env, Tree: tree}, nil
}

// Encode serializes a Value back to the on-disk byte form. It reuses
// Envelope.Compressed when present (preserving the original payload
// byte-for-byte) and recompresses the inflated buffer otherwise.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("BDHb: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("BDHb: Encode expected Value or *Value, got %T", value)
	}
	return heap.EncodeEnvelope(v.Envelope)
}

// Validate reports structural issues with payload.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)}
	env, err := heap.DecodeEnvelope(payload)
	if err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "bdhb.envelope.invalid",
			Message:  fmt.Sprintf("BDHb envelope could not be parsed: %v", err),
			Location: loc,
		}}
	}
	if _, err := heap.Walk(env.Content); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "bdhb.tagstream.invalid",
			Message:  fmt.Sprintf("BDHb tag stream could not be walked: %v", err),
			Location: loc,
		}}
	}
	return nil
}
