package heap

import (
	"encoding/binary"
	"fmt"
)

// readU124 reads pylabview's U124 unsigned variable-size encoding from
// buf. Layout (LVmisc.py:397-405):
//
//   - 1 byte unsigned. Value < 254 → that's the result.
//   - 0xFF (255) → next 2 bytes BE uint16 are the result.
//   - 0xFE (254) → next 4 bytes BE uint32 are the result.
//
// Returns (value, bytes consumed, error).
func readU124(buf []byte) (uint32, int, error) {
	if len(buf) < 1 {
		return 0, 0, fmt.Errorf("heap: u124 needs at least 1 byte, have %d", len(buf))
	}
	tag := buf[0]
	switch tag {
	case 0xFF:
		if len(buf) < 3 {
			return 0, 0, fmt.Errorf("heap: u124 escape 0xFF needs 3 bytes, have %d", len(buf))
		}
		return uint32(binary.BigEndian.Uint16(buf[1:3])), 3, nil
	case 0xFE:
		if len(buf) < 5 {
			return 0, 0, fmt.Errorf("heap: u124 escape 0xFE needs 5 bytes, have %d", len(buf))
		}
		return binary.BigEndian.Uint32(buf[1:5]), 5, nil
	default:
		return uint32(tag), 1, nil
	}
}

// readS124 reads pylabview's S124 signed variable-size encoding from
// buf. Layout (LVmisc.py:376-384):
//
//   - 1 byte signed. Value not in {-128, -127} → that's the result.
//   - -128 (0x80) → next 2 bytes BE int16 are the result.
//   - -127 (0x81) → next 4 bytes BE int32 are the result.
//
// Note: pylabview's prepare logic shows S124 prefers the smaller form,
// so genuine -128 / -127 byte values cannot appear in well-formed
// payloads. The reader still accepts them as escape codes only.
func readS124(buf []byte) (int32, int, error) {
	if len(buf) < 1 {
		return 0, 0, fmt.Errorf("heap: s124 needs at least 1 byte, have %d", len(buf))
	}
	tag := int8(buf[0])
	switch tag {
	case -128:
		if len(buf) < 3 {
			return 0, 0, fmt.Errorf("heap: s124 escape -128 needs 3 bytes, have %d", len(buf))
		}
		return int32(int16(binary.BigEndian.Uint16(buf[1:3]))), 3, nil
	case -127:
		if len(buf) < 5 {
			return 0, 0, fmt.Errorf("heap: s124 escape -127 needs 5 bytes, have %d", len(buf))
		}
		return int32(binary.BigEndian.Uint32(buf[1:5])), 5, nil
	default:
		return int32(tag), 1, nil
	}
}

// readS24 reads pylabview's S24 signed variable-size encoding from buf.
// Layout (LVmisc.py:357-363):
//
//   - 2 bytes BE int16. Value != -0x8000 → that's the result.
//   - 2 bytes BE int16 == -0x8000 → next 4 bytes BE int32 are the result.
//
// Returns (value, bytes consumed, error).
func readS24(buf []byte) (int32, int, error) {
	if len(buf) < 2 {
		return 0, 0, fmt.Errorf("heap: s24 needs at least 2 bytes, have %d", len(buf))
	}
	v := int16(binary.BigEndian.Uint16(buf[:2]))
	if v != -0x8000 {
		return int32(v), 2, nil
	}
	if len(buf) < 6 {
		return 0, 0, fmt.Errorf("heap: s24 escape -0x8000 needs 6 bytes, have %d", len(buf))
	}
	return int32(binary.BigEndian.Uint32(buf[2:6])), 6, nil
}
