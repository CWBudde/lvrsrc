package lvmeta

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// noNameOffsetSentinel is the wire-format marker for sections without a name
// entry in the name table. It matches rsrcwire's unexported noNameOffset.
const noNameOffsetSentinel = ^uint32(0)

// SetDescription sets the VI description, routing through the STRG resource.
//
// It is a convenience over (Mutator{}).SetDescription and therefore runs in
// lenient (non-strict) mode.
func SetDescription(f *lvrsrc.File, desc string) error {
	return Mutator{}.SetDescription(f, desc)
}

// SetDescription sets the VI description.
//
// Behavior by existing STRG section count:
//   - 1: update the section's payload in place via the generic typed
//     mutation pipeline; all Tier 2 gates (codec tier, WriteVersions,
//     post-edit validate, and — in strict mode — new-warning rejection)
//     apply.
//   - 0: append a new STRG block with a single section whose Index is 0,
//     Name is empty, and NameOffset is the no-name sentinel.
//   - >1: reject with ErrTargetAmbiguous. There is no automatic selection
//     rule yet; corpus evidence has not justified one.
//
// Caller-provided bytes are preserved verbatim: no newline normalization,
// trimming, or charset transcoding. Empty descriptions produce a valid
// zero-length STRG payload (a 4-byte size prefix of 0).
func (m Mutator) SetDescription(f *lvrsrc.File, desc string) error {
	if f == nil {
		return &MutationError{FourCC: strg.FourCC, Cause: ErrNilFile}
	}

	refs := findSectionsByType(f, strg.FourCC)
	switch len(refs) {
	case 1:
		return m.applyTypedEdit(f, strg.FourCC, func(any) (any, error) {
			return strg.Value{Text: desc}, nil
		})
	case 0:
		return m.createSTRGSection(f, desc)
	default:
		return &MutationError{
			FourCC: strg.FourCC,
			Cause:  ErrTargetAmbiguous,
			Err:    fmt.Errorf("%d STRG sections present", len(refs)),
		}
	}
}

// createSTRGSection appends a new STRG block with one section whose payload
// encodes desc. It enforces the same Tier 2 gates as applyTypedEdit
// (tier, WriteVersions, post-edit validate, strict-mode warning policy)
// before mutating f.
func (m Mutator) createSTRGSection(f *lvrsrc.File, desc string) error {
	codec := m.effectiveRegistry().Lookup(strg.FourCC)
	capability := codec.Capability()
	base := &MutationError{FourCC: strg.FourCC}

	if capability.Safety != codecs.SafetyTier2 {
		e := *base
		e.Cause = ErrUnsafeForTier2
		return &e
	}
	ctx := contextFromFile(f)
	if !capability.WriteVersions.Contains(ctx.FileVersion) {
		e := *base
		e.Cause = ErrUnsupportedVersion
		return &e
	}

	payload, err := codec.Encode(ctx, strg.Value{Text: desc})
	if err != nil {
		e := *base
		e.Cause = ErrCodecEncode
		e.Err = err
		return &e
	}

	issues := codec.Validate(ctx, payload)
	for _, iss := range issues {
		if iss.Severity == validate.SeverityError {
			e := *base
			e.Cause = ErrPostEditValidation
			return &e
		}
	}
	if m.Strict {
		for _, iss := range issues {
			if iss.Severity == validate.SeverityWarning {
				e := *base
				e.Cause = ErrPostEditWarning
				return &e
			}
		}
	}

	baseline := captureStructuralBaseline(f)

	origBlocks := f.Blocks
	f.Blocks = append(f.Blocks, lvrsrc.Block{
		Type:                 strg.FourCC,
		SectionCountMinusOne: 0,
		Sections: []lvrsrc.Section{
			{
				Index:      0,
				NameOffset: noNameOffsetSentinel,
				Name:       "",
				Payload:    payload,
			},
		},
	})
	if err := m.runStructuralCheck(f, strg.FourCC, baseline); err != nil {
		f.Blocks = origBlocks
		return err
	}
	return nil
}
