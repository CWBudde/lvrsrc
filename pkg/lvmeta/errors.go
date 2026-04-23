package lvmeta

import (
	"errors"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
)

// Sentinel errors returned (wrapped in *MutationError) by the Tier 2 mutation
// pipeline. Callers can branch on these with errors.Is.
var (
	// ErrNilFile is returned when a mutator is invoked on a nil *lvrsrc.File.
	ErrNilFile = errors.New("lvmeta: nil file")

	// ErrTargetMissing is returned when no section matches the requested FourCC.
	ErrTargetMissing = errors.New("lvmeta: target resource not found")

	// ErrTargetAmbiguous is returned when more than one section matches the
	// requested FourCC and no disambiguating rule applies.
	ErrTargetAmbiguous = errors.New("lvmeta: target resource is ambiguous")

	// ErrUnsafeForTier2 is returned when the registered codec's Capability
	// does not declare SafetyTier2.
	ErrUnsafeForTier2 = errors.New("lvmeta: codec is not Tier 2 editable")

	// ErrUnsupportedVersion is returned when the file's format version is
	// outside the codec's declared WriteVersions range.
	ErrUnsupportedVersion = errors.New("lvmeta: file version not in codec WriteVersions")

	// ErrCodecDecode is returned when the codec rejects the existing payload.
	ErrCodecDecode = errors.New("lvmeta: codec decode failed")

	// ErrMutation is returned when the caller-supplied mutate callback reports
	// an error; the original payload is left untouched.
	ErrMutation = errors.New("lvmeta: mutation callback failed")

	// ErrCodecEncode is returned when the codec refuses to encode the mutated
	// value (for example, out-of-range fields).
	ErrCodecEncode = errors.New("lvmeta: codec encode failed")

	// ErrPostEditValidation is returned when the encoded payload fails the
	// codec's Validate with at least one severity-error issue.
	ErrPostEditValidation = errors.New("lvmeta: post-edit validation reported errors")

	// ErrPostEditWarning is returned, only in strict mode, when the encoded
	// payload introduces a codec warning that was not present before the edit.
	ErrPostEditWarning = errors.New("lvmeta: post-edit validation introduced a new warning")

	// ErrNameTooLong is returned when a proposed name exceeds the Pascal-string
	// byte-length limit used by the RSRC name table (255 bytes).
	ErrNameTooLong = errors.New("lvmeta: name exceeds Pascal-string length limit")

	// ErrStructuralValidation is returned when the post-edit safety gate
	// detects that the mutated file would fail structural validation after
	// serialization (for example: unresolvable name offsets, block count
	// mismatches, or serializer rejection of the mutated state).
	ErrStructuralValidation = errors.New("lvmeta: post-edit structural validation failed")
)

// MutationError carries FourCC and in-file offset context for a failed Tier 2
// mutation, in addition to a sentinel Cause that can be matched with errors.Is.
type MutationError struct {
	// FourCC identifies the targeted resource type.
	FourCC codecs.FourCC
	// Offset is the on-disk DataOffset of the targeted section when a single
	// target was located; 0 otherwise.
	Offset uint32
	// SectionIndex is the in-file Section.Index for the targeted section when
	// a single target was located; zero otherwise.
	SectionIndex int32
	// Cause is a sentinel error from the list above (ErrTargetMissing, etc.).
	Cause error
	// Err is an optional underlying error (codec-level failure, mutate
	// callback error, etc.).
	Err error
}

// Error implements error.
func (e *MutationError) Error() string {
	switch {
	case e.FourCC != "" && e.Offset != 0 && e.Err != nil:
		return fmt.Sprintf("%s: %q at offset %d: %v", e.Cause, e.FourCC, e.Offset, e.Err)
	case e.FourCC != "" && e.Err != nil:
		return fmt.Sprintf("%s: %q: %v", e.Cause, e.FourCC, e.Err)
	case e.FourCC != "" && e.Offset != 0:
		return fmt.Sprintf("%s: %q at offset %d", e.Cause, e.FourCC, e.Offset)
	case e.FourCC != "":
		return fmt.Sprintf("%s: %q", e.Cause, e.FourCC)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Cause, e.Err)
	default:
		return e.Cause.Error()
	}
}

// Unwrap lets errors.Is match either the sentinel Cause or the underlying Err.
func (e *MutationError) Unwrap() []error {
	if e.Err != nil {
		return []error{e.Cause, e.Err}
	}
	return []error{e.Cause}
}
