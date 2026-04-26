package linkobj

import (
	"encoding/binary"
	"fmt"
)

// VIToLib is the typed projection of a VILB LinkObjRef. pylabview's
// LinkObjVIToLib (LVlinkinfo.py:2186) reduces to: ident +
// parseBasicLinkSaveInfo. After the primary path that the surrounding
// codec extracts, the on-disk wire layout is exactly:
//
//	  linkSaveFlag (u32 BE)            // BasicLinkSaveInfo, v8.6+
//
// VILB carries no secondary path.
type VIToLib struct {
	IdentValue   string // "VILB"
	LinkSaveFlag uint32
}

// Kind reports KindVIToLib.
func (VIToLib) Kind() LinkKind { return KindVIToLib }

// Ident returns the wire ident.
func (v VIToLib) Ident() string { return v.IdentValue }

func decodeVIToLib(ident string, body, secondaryRaw []byte) (VIToLib, error) {
	if len(secondaryRaw) != 0 {
		return VIToLib{}, fmt.Errorf("VILB: unexpected secondary path (%d bytes)", len(secondaryRaw))
	}
	if len(body) != 4 {
		return VIToLib{}, fmt.Errorf("VILB: body must be exactly 4 bytes (linkSaveFlag); have %d", len(body))
	}
	return VIToLib{
		IdentValue:   ident,
		LinkSaveFlag: binary.BigEndian.Uint32(body[:4]),
	}, nil
}

func encodeVIToLib(v VIToLib) (body, secondaryRaw []byte, err error) {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, v.LinkSaveFlag)
	return out, nil, nil
}
