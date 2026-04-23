package lvmeta

import (
	"bytes"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// structuralBaseline captures the set of structural-error codes present
// in f before a mutation is applied. It serializes f, re-parses, and
// runs the validator; nil means capture failed (serialize or re-parse
// errored) and the post-edit gate cannot reliably flag new issues.
type structuralBaseline struct {
	captured bool
	codes    map[string]struct{}
}

func captureStructuralBaseline(f *lvrsrc.File) structuralBaseline {
	codes, ok := collectStructuralErrorCodes(f)
	return structuralBaseline{captured: ok, codes: codes}
}

// collectStructuralErrorCodes returns the set of severity-error codes the
// validator reports on f's round-trip. ok is false if the file cannot be
// serialized or re-parsed at all.
func collectStructuralErrorCodes(f *lvrsrc.File) (map[string]struct{}, bool) {
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, false
	}
	parsed, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		return nil, false
	}
	codes := map[string]struct{}{}
	for _, iss := range parsed.Validate() {
		if iss.Severity == lvrsrc.SeverityError {
			codes[iss.Code] = struct{}{}
		}
	}
	return codes, true
}

// runStructuralCheck is the Tier 2 post-edit safety gate.
//
// It serializes f, re-parses the output, and runs the structural validator
// on the round-tripped file. The gate fails only for *new* error-severity
// codes — errors already present before the mutation are tolerated. When
// baseline capture failed (synthetic in-memory fixtures without valid
// headers), the gate is skipped because we cannot distinguish pre-existing
// from edit-induced breakage.
//
// Warnings never fail the gate; Tier 2 strict-mode warning policy is
// enforced separately at the codec level inside applyTypedEdit.
func (m Mutator) runStructuralCheck(f *lvrsrc.File, fourCC codecs.FourCC, baseline structuralBaseline) error {
	if !baseline.captured {
		return nil
	}

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return &MutationError{
			FourCC: fourCC,
			Cause:  ErrStructuralValidation,
			Err:    fmt.Errorf("serialize: %w", err),
		}
	}

	parsed, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		return &MutationError{
			FourCC: fourCC,
			Cause:  ErrStructuralValidation,
			Err:    fmt.Errorf("re-parse: %w", err),
		}
	}

	for _, iss := range parsed.Validate() {
		if iss.Severity != lvrsrc.SeverityError {
			continue
		}
		if _, existed := baseline.codes[iss.Code]; existed {
			continue
		}
		return &MutationError{
			FourCC: fourCC,
			Cause:  ErrStructuralValidation,
			Err:    fmt.Errorf("%s: %s", iss.Code, iss.Message),
		}
	}
	return nil
}
