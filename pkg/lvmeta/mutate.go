package lvmeta

import (
	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// applyTypedEdit performs the common Tier 2 edit flow for a single FourCC
// target:
//
//  1. Locate exactly one section whose block Type matches fourCC; reject
//     missing (ErrTargetMissing) or duplicated (ErrTargetAmbiguous) targets.
//  2. Look up the codec in the effective registry and enforce
//     Capability.Safety == SafetyTier2.
//  3. Derive a codecs.Context from the file and enforce
//     Capability.WriteVersions.Contains(ctx.FileVersion).
//  4. Decode the current payload; on failure return ErrCodecDecode.
//  5. Apply mutate; propagate callback errors as ErrMutation.
//  6. Re-encode; on failure return ErrCodecEncode.
//  7. Run codec.Validate on the encoded payload; return
//     ErrPostEditValidation on any severity-error issue, and (Strict only)
//     ErrPostEditWarning on any warning whose code is new relative to the
//     pre-edit payload.
//  8. Only after every gate passes, swap the section's Payload with the
//     encoded bytes. The original payload is preserved byte-for-byte on
//     every failure path.
//
// All failures wrap the sentinel in *MutationError with FourCC + offset
// context for callers.
func (m Mutator) applyTypedEdit(f *lvrsrc.File, fourCC codecs.FourCC, mutate func(any) (any, error)) error {
	ref, found, locErr := requireSingleSectionByType(f, fourCC)
	if locErr != nil {
		return &MutationError{FourCC: fourCC, Cause: ErrTargetAmbiguous, Err: locErr}
	}
	if !found {
		return &MutationError{FourCC: fourCC, Cause: ErrTargetMissing}
	}

	baseline := captureStructuralBaseline(f)

	section := &f.Blocks[ref.BlockIndex].Sections[ref.SectionIndex]
	baseErr := &MutationError{
		FourCC:       fourCC,
		Offset:       section.DataOffset,
		SectionIndex: section.Index,
	}
	fail := func(cause, err error) error {
		e := *baseErr
		e.Cause = cause
		e.Err = err
		return &e
	}

	codec := m.effectiveRegistry().Lookup(fourCC)
	cap := codec.Capability()
	if cap.Safety != codecs.SafetyTier2 {
		return fail(ErrUnsafeForTier2, nil)
	}

	ctx := contextFromFile(f)
	if !cap.WriteVersions.Contains(ctx.FileVersion) {
		return fail(ErrUnsupportedVersion, nil)
	}

	var baselineWarnings map[string]struct{}
	if m.Strict {
		baselineWarnings = warningCodeSet(codec.Validate(ctx, section.Payload))
	}

	value, err := codec.Decode(ctx, section.Payload)
	if err != nil {
		return fail(ErrCodecDecode, err)
	}

	newValue, err := mutate(value)
	if err != nil {
		return fail(ErrMutation, err)
	}

	newPayload, err := codec.Encode(ctx, newValue)
	if err != nil {
		return fail(ErrCodecEncode, err)
	}

	issues := codec.Validate(ctx, newPayload)
	for _, iss := range issues {
		if iss.Severity == validate.SeverityError {
			return fail(ErrPostEditValidation, nil)
		}
	}
	if m.Strict {
		for _, iss := range issues {
			if iss.Severity != validate.SeverityWarning {
				continue
			}
			if _, existed := baselineWarnings[iss.Code]; !existed {
				return fail(ErrPostEditWarning, nil)
			}
		}
	}

	oldPayload := section.Payload
	section.Payload = newPayload
	if err := m.runStructuralCheck(f, fourCC, baseline); err != nil {
		section.Payload = oldPayload
		return err
	}
	return nil
}

func warningCodeSet(issues []validate.Issue) map[string]struct{} {
	out := make(map[string]struct{}, len(issues))
	for _, iss := range issues {
		if iss.Severity == validate.SeverityWarning {
			out[iss.Code] = struct{}{}
		}
	}
	return out
}
