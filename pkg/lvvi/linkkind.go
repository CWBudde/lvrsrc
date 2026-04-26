package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/linkobj"

// LinkKind is the stable Go-named identifier for a LinkObjRef subclass.
// It mirrors the values exposed by internal/codecs/linkobj so callers do
// not need to import the internal package. New kinds may be appended;
// switching on a LinkKind should always include a default case for
// forward compatibility.
type LinkKind int

const (
	// LinkKindUnknown is returned when the on-disk 4-byte ident has no
	// recognised LinkKind mapping.
	LinkKindUnknown LinkKind = LinkKind(linkobj.KindUnknown)

	// Selected concrete kinds — the rest are accessible through their
	// 4-byte ident plus LookupLinkKind. Pylabview's full set has ~115
	// entries; we surface the ones a UI is most likely to switch on.
	LinkKindTypeDefToCCLink LinkKind = LinkKind(linkobj.KindTypeDefToCCLink)
	LinkKindVIToLib         LinkKind = LinkKind(linkobj.KindVIToLib)
	LinkKindIUseToVILink    LinkKind = LinkKind(linkobj.KindIUseToVILink)
	LinkKindVIToCCLink      LinkKind = LinkKind(linkobj.KindVIToCCLink)
)

// LookupLinkKind returns the LinkKind for an on-disk 4-byte ident, or
// LinkKindUnknown if the ident isn't recognised.
func LookupLinkKind(ident string) LinkKind {
	return LinkKind(linkobj.LookupKind(ident))
}

// LinkKindIdent returns the canonical 4-byte ident for a LinkKind, or
// the empty string for LinkKindUnknown.
func LinkKindIdent(k LinkKind) string {
	return linkobj.CanonicalIdent(linkobj.LinkKind(k))
}

// LinkKindDescription returns a short human-readable label for a
// LinkKind (e.g. "TypeDef → CustCtl"). Returns "unknown link" for
// LinkKindUnknown.
func LinkKindDescription(k LinkKind) string {
	return linkobj.Description(linkobj.LinkKind(k))
}
