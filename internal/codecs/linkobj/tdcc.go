package linkobj

import (
	"encoding/binary"
	"fmt"
)

// TypeDefToCCLink is the typed projection of a TDCC (or list-aliased
// LVCC under FPHP) LinkObjRef. It corresponds to pylabview's
// LinkObjTypeDefToCCLink (LVlinkinfo.py:1942), whose parseRSRCData
// reduces to: ident + HeapToVILinkSaveInfo. After the primary path —
// which the surrounding LIfp/LIbd/LIvi codec has already extracted into
// the entry's PrimaryPath — the wire layout is:
//
//	  linkSaveFlag (u32 BE)            // BasicLinkSaveInfo, v8.6+
//	  typedLinkTD.index (varU2p2)      // TypedLinkSaveInfo, v8.0+
//	  vilinkref (1B or 25B)            // VILinkRefInfo, v14+
//	  typedLinkFlags (u32 BE)          // TypedLinkSaveInfo, v12+
//	  linkOffsetCount (u32 BE)         // LinkOffsetList, v8.2+
//	  offsets[count] (each u32 BE)
//	  viLSPathRef (PTH0)               // HeapToVILinkSaveInfo, v8.2+
//
// The viLSPathRef is *not* part of body; it is the SecondaryPath the
// surrounding codec extracted via splitTailAndSecondary.
type TypeDefToCCLink struct {
	IdentValue     string // "TDCC" or, in some FPHP list contexts, "LVCC"
	LinkSaveFlag   uint32
	TypeDescID     uint32
	TypeDescIDWide bool // true ⇒ TypeDescID was encoded with the 4-byte u2p2 form
	VILinkRef      VILinkRef
	TypedLinkFlags uint32
	Offsets        []uint32
	ViLSPathRef    []byte // raw PTH0 bytes from the surrounding SecondaryPath; nil if absent
}

// Kind reports KindTypeDefToCCLink.
func (TypeDefToCCLink) Kind() LinkKind { return KindTypeDefToCCLink }

// Ident returns the wire ident (typically "TDCC").
func (t TypeDefToCCLink) Ident() string { return t.IdentValue }

func decodeTypeDefToCC(ident string, body, secondaryRaw []byte) (TypeDefToCCLink, error) {
	t := TypeDefToCCLink{IdentValue: ident}
	off := 0
	if len(body) < 4 {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: body too short for linkSaveFlag (have %d, need 4)", len(body))
	}
	t.LinkSaveFlag = binary.BigEndian.Uint32(body[off : off+4])
	off += 4

	val, n, err := readVarSizeU2p2(body[off:])
	if err != nil {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: typeDescID: %w", err)
	}
	t.TypeDescID = val
	t.TypeDescIDWide = n == 4
	off += n

	vil, n, err := decodeVILinkRef(body[off:])
	if err != nil {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: %w", err)
	}
	t.VILinkRef = vil
	off += n

	if len(body[off:]) < 4 {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: body truncated before typedLinkFlags (have %d, need 4)", len(body[off:]))
	}
	t.TypedLinkFlags = binary.BigEndian.Uint32(body[off : off+4])
	off += 4

	if len(body[off:]) < 4 {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: body truncated before offset count (have %d, need 4)", len(body[off:]))
	}
	count := binary.BigEndian.Uint32(body[off : off+4])
	off += 4
	const maxOffsets = 1 << 16 // sanity cap; pylabview uses typedesc_list_limit (8000)
	if count > maxOffsets {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: offset count %d exceeds sanity cap %d", count, maxOffsets)
	}
	if len(body[off:]) < int(count)*4 {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: body truncated for %d offsets (have %d bytes)", count, len(body[off:]))
	}
	t.Offsets = make([]uint32, count)
	for i := range t.Offsets {
		t.Offsets[i] = binary.BigEndian.Uint32(body[off : off+4])
		off += 4
	}

	if off != len(body) {
		return TypeDefToCCLink{}, fmt.Errorf("TDCC: %d trailing body bytes after offsets", len(body)-off)
	}

	// secondaryRaw, if present, IS the viLSPathRef. We keep it as raw
	// bytes since the lifp/libd codec is the authoritative PTH0 parser
	// (and we want byte-for-byte round-trip even when secondaryRaw is
	// PTH0 with declaredLen=0, i.e. an "empty path" placeholder).
	if len(secondaryRaw) > 0 {
		t.ViLSPathRef = cloneBytes(secondaryRaw)
	}
	return t, nil
}

func encodeTypeDefToCC(t TypeDefToCCLink) (body, secondaryRaw []byte, err error) {
	out := make([]byte, 0, 4+4+1+4+4+len(t.Offsets)*4)
	out = appendU32(out, t.LinkSaveFlag)
	if t.TypeDescIDWide {
		// Force the wide form even if the value would fit in 15 bits, so
		// that a corpus value that was originally encoded wide
		// re-encodes wide. The high bit is set on the wide form.
		out = append(out, writeVarSizeU2p2Wide(t.TypeDescID)...)
	} else {
		if t.TypeDescID > 0x7fff {
			return nil, nil, fmt.Errorf("TDCC: typeDescID %d exceeds 15-bit form but TypeDescIDWide is false", t.TypeDescID)
		}
		out = append(out, writeVarSizeU2p2(t.TypeDescID)...)
	}
	out = append(out, encodeVILinkRef(t.VILinkRef)...)
	out = appendU32(out, t.TypedLinkFlags)
	out = appendU32(out, uint32(len(t.Offsets)))
	for _, o := range t.Offsets {
		out = appendU32(out, o)
	}
	return out, cloneBytes(t.ViLSPathRef), nil
}

// writeVarSizeU2p2Wide forces the 4-byte form regardless of value. Used
// only when round-tripping a corpus value that was originally encoded
// wide.
func writeVarSizeU2p2Wide(val uint32) []byte {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], val|0x80000000)
	return b[:]
}

func appendU32(dst []byte, v uint32) []byte {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	return append(dst, b[:]...)
}
