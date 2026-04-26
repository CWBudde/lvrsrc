package linkobj

import (
	"encoding/binary"
	"fmt"
)

// LinkTarget is the typed projection of a LinkObjRef. Concrete
// implementations cover individual subclasses; OpaqueTarget is the
// round-trip-safe fallback for idents that haven't been ported yet.
type LinkTarget interface {
	// Kind returns the stable LinkKind of this target.
	Kind() LinkKind
	// Ident returns the 4-byte wire ident (e.g. "TDCC", "VILB").
	Ident() string
}

// OpaqueTarget preserves the raw post-primary-path bytes of a link
// payload exactly as they appear on disk. It is used for any LinkKind
// without a typed parser, and guarantees byte-for-byte round-trip via
// Encode.
type OpaqueTarget struct {
	IdentValue   string // 4-byte wire ident
	Body         []byte // post-primary-path bytes (the lifp/libd "Tail")
	SecondaryRaw []byte // optional viLSPathRef bytes; nil if absent
}

// Kind reports the LinkKind associated with the ident, falling back to
// KindUnknown if the ident isn't registered.
func (o OpaqueTarget) Kind() LinkKind { return LookupKind(o.IdentValue) }

// Ident returns the wire ident.
func (o OpaqueTarget) Ident() string { return o.IdentValue }

// VILinkRef captures the v14+ VILinkRefInfo block. flagBt is the leading
// byte; when it equals 0xff the legacy structured form follows in the
// remaining fields. References:
// references/pylabview/pylabview/LVlinkinfo.py:208-260.
type VILinkRef struct {
	FlagBt byte
	// FieldA, LibVersion, Field4 are unpacked from FlagBt when FlagBt != 0xff
	// and decoded from explicit fields otherwise.
	FieldA     byte
	LibVersion uint8 // 5 bits when packed in flagBt
	Field4     byte  // 2 bits when packed in flagBt; 4 bytes when explicit (truncated to byte for now)
	// Explicit fields used when FlagBt == 0xff.
	Field4Raw     uint32
	LibVersionRaw uint64
	FieldB        [4]byte
	FieldC        [4]byte
	FieldD        int32
	HasExplicit   bool
}

// decodeVILinkRef parses the v14+ VILinkRefInfo. Returns the struct and
// the number of bytes consumed.
func decodeVILinkRef(buf []byte) (VILinkRef, int, error) {
	if len(buf) < 1 {
		return VILinkRef{}, 0, fmt.Errorf("vilinkref: need 1 byte for flagBt")
	}
	v := VILinkRef{FlagBt: buf[0]}
	if v.FlagBt != 0xff {
		v.FieldA = v.FlagBt & 0x01
		v.LibVersion = (v.FlagBt >> 1) & 0x1f
		v.Field4 = (v.FlagBt >> 6) & 0x03
		return v, 1, nil
	}
	// Explicit form — 4-byte field4, 8-byte libVersion, 4-byte fieldB,
	// 4-byte fieldC, 4-byte signed fieldD. 25 bytes total including the
	// flagBt byte we already consumed.
	if len(buf) < 25 {
		return VILinkRef{}, 0, fmt.Errorf("vilinkref: explicit form needs 25 bytes, have %d", len(buf))
	}
	v.HasExplicit = true
	v.Field4Raw = binary.BigEndian.Uint32(buf[1:5])
	v.LibVersionRaw = binary.BigEndian.Uint64(buf[5:13])
	copy(v.FieldB[:], buf[13:17])
	copy(v.FieldC[:], buf[17:21])
	v.FieldD = int32(binary.BigEndian.Uint32(buf[21:25]))
	return v, 25, nil
}

// encodeVILinkRef inverts decodeVILinkRef.
func encodeVILinkRef(v VILinkRef) []byte {
	if !v.HasExplicit {
		// Compact form: a single byte derived from packed fields. We
		// honour the original FlagBt verbatim so round-trip stays exact
		// even if a caller has mutated FieldA/LibVersion/Field4
		// inconsistently with FlagBt — packed mutations are not yet
		// supported.
		return []byte{v.FlagBt}
	}
	out := make([]byte, 25)
	out[0] = 0xff
	binary.BigEndian.PutUint32(out[1:5], v.Field4Raw)
	binary.BigEndian.PutUint64(out[5:13], v.LibVersionRaw)
	copy(out[13:17], v.FieldB[:])
	copy(out[17:21], v.FieldC[:])
	binary.BigEndian.PutUint32(out[21:25], uint32(v.FieldD))
	return out
}

// Decode parses the post-primary-path payload of a LinkObjRef. ident is
// the entry's 4-byte LinkType (which is *also* the LinkObj's wire ident
// per pylabview's parseRSRCData), body is the bytes between the primary
// PTH0 and the next entry/footer (or before any extracted secondary
// PTH0), and secondaryRaw is the raw bytes of an optional viLSPathRef
// that the surrounding codec already separated. Either, both, or neither
// argument may be empty/nil depending on the LinkKind.
//
// The returned LinkTarget always re-encodes byte-for-byte.
func Decode(ident string, body, secondaryRaw []byte) (LinkTarget, error) {
	switch ident {
	case "TDCC", "LVCC":
		return decodeTypeDefToCC(ident, body, secondaryRaw)
	case "VILB":
		return decodeVIToLib(ident, body, secondaryRaw)
	default:
		return OpaqueTarget{
			IdentValue:   ident,
			Body:         cloneBytes(body),
			SecondaryRaw: cloneBytes(secondaryRaw),
		}, nil
	}
}

// Encode is the inverse of Decode. It returns the post-primary-path body
// bytes plus the raw secondary-path bytes (or nil), in their original
// on-disk form.
func Encode(t LinkTarget) (body, secondaryRaw []byte, err error) {
	switch v := t.(type) {
	case OpaqueTarget:
		return cloneBytes(v.Body), cloneBytes(v.SecondaryRaw), nil
	case TypeDefToCCLink:
		return encodeTypeDefToCC(v)
	case VIToLib:
		return encodeVIToLib(v)
	case nil:
		return nil, nil, fmt.Errorf("linkobj.Encode: nil target")
	default:
		return nil, nil, fmt.Errorf("linkobj.Encode: unsupported concrete type %T", t)
	}
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
