package linkobj

import (
	"encoding/binary"
	"fmt"
)

// readVarSizeU2p2 reads pylabview's "U2p2" variable-size field. The first
// 16-bit big-endian word's high bit is the size flag: clear → just the
// 15-bit value; set → ((word & 0x7fff) << 16) | next 16 bits.
//
// Reference: references/pylabview/pylabview/LVmisc.py:336.
func readVarSizeU2p2(buf []byte) (val uint32, consumed int, err error) {
	if len(buf) < 2 {
		return 0, 0, fmt.Errorf("varsize u2p2: need 2 bytes, have %d", len(buf))
	}
	word := binary.BigEndian.Uint16(buf[:2])
	if word&0x8000 == 0 {
		return uint32(word), 2, nil
	}
	if len(buf) < 4 {
		return 0, 0, fmt.Errorf("varsize u2p2: extended form needs 4 bytes, have %d", len(buf))
	}
	hi := uint32(word & 0x7fff)
	lo := uint32(binary.BigEndian.Uint16(buf[2:4]))
	return (hi << 16) | lo, 4, nil
}

// writeVarSizeU2p2 emits the inverse of readVarSizeU2p2.
//
// Reference: references/pylabview/pylabview/LVmisc.py:348.
func writeVarSizeU2p2(val uint32) []byte {
	if val <= 0x7fff {
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(val))
		return b[:]
	}
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], val|0x80000000)
	return b[:]
}
