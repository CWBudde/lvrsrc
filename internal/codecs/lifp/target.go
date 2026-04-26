package lifp

import "github.com/CWBudde/lvrsrc/internal/codecs/linkobj"

// Target decodes the entry's post-primary-path payload into a typed
// linkobj.LinkTarget. The decode is lazy: round-trip serialization
// continues to use Tail and SecondaryPath as the byte-authoritative
// source, so calling Target() is safe and does not affect Encode output.
//
// Targets for ports without a typed parser fall back to
// linkobj.OpaqueTarget (which itself round-trips byte-for-byte).
func (e Entry) Target() (linkobj.LinkTarget, error) {
	var secondaryRaw []byte
	if e.SecondaryPath != nil {
		secondaryRaw = e.SecondaryPath.Raw
	}
	return linkobj.Decode(e.LinkType, e.Tail, secondaryRaw)
}
