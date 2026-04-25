// Package pthx decodes and encodes LabVIEW path references — the
// `PTH0` / `PTH1` / `PTH2` data structures embedded inside link-info
// blocks (`LIfp`, `LIbd`, `LIvi`) and elsewhere.
//
// LabVIEW ships two layouts. Selection is by leading FourCC:
//
//   - `PTH0` (LVPath0) — 1-byte length per component, 2-byte tpval +
//     2-byte component count.
//   - `PTH1` / `PTH2` (LVPath1) — 2-byte length per component, 4-byte
//     tpident (`"unc "` / `"!pth"` / `"abs "` / `"rel "`).
//
// Both share a `[ident:4][totlen:4]` envelope. PTH0 has a "phony"
// 8-byte form (`PTH0 + 4 zero bytes`) that LabVIEW writes when the path
// stores nothing — pylabview calls it the `canZeroFill` case. The
// codec preserves it so re-encoding produces the same on-disk bytes.
//
// References:
//   - pylabview LVclasses.py:94 (LVPath1) and :159 (LVPath0).
//   - pylabview LVlinkinfo.py:66-78 (parsePathRef — the variant
//     dispatch this codec mirrors).
//
// Unlike the other codecs in `internal/codecs/`, pthx does not
// implement `codecs.ResourceCodec` — paths are never registered as
// top-level resources; they are read out of larger payloads. Decode
// returns a `Value` plus the byte count it consumed, so callers can
// chain it through composite blocks.
package pthx

import (
	"encoding/binary"
	"fmt"
)

// Idents recognised by Decode.
const (
	IdentPTH0 = "PTH0"
	IdentPTH1 = "PTH1"
	IdentPTH2 = "PTH2"
)

// Path-type identifiers (PTH1/PTH2 only).
const (
	TPIdentAbsolute = "abs "
	TPIdentRelative = "rel "
	TPIdentUNC      = "unc "
	TPIdentNotPath  = "!pth"
)

// Value is the decoded form of a path reference. The relevant subset of
// fields depends on Ident:
//
//   - PTH0: TPVal + Components (1-byte-length-prefix Pascal strings).
//     ZeroFill marks the special 8-byte phony form.
//   - PTH1 / PTH2: TPIdent + Components (2-byte-length-prefix strings).
//
// Components are kept as []byte rather than string so the codec stays
// transparent about LabVIEW's text encoding (caller's responsibility).
type Value struct {
	// Ident is the leading FourCC, e.g. "PTH0", "PTH1", "PTH2".
	Ident string

	// TPVal is the PTH0 type-value field (offsets 8..9). pylabview
	// observes only 0 and 1 in the wild but documents the field as
	// 0..3. Meaningful only when Ident == "PTH0".
	TPVal uint16

	// TPIdent is the PTH1/PTH2 type-identifier 4-character code,
	// e.g. "abs ", "rel ", "unc ", "!pth". Meaningful only when
	// Ident is "PTH1" or "PTH2".
	TPIdent string

	// Components are the path segments in their on-disk order.
	Components [][]byte

	// ZeroFill marks the PTH0 phony form: a path whose envelope
	// contains only the FourCC ident plus a zero totlen, with no
	// further bytes. pylabview calls this `canZeroFill`.
	ZeroFill bool
}

// IsPTH0 reports whether Value uses the PTH0 layout.
func (v Value) IsPTH0() bool { return v.Ident == IdentPTH0 }

// IsPTH1 reports whether Value uses the PTH1/PTH2 layout.
func (v Value) IsPTH1() bool { return v.Ident == IdentPTH1 || v.Ident == IdentPTH2 }

// IsAbsolute reports whether the path is tagged as absolute (PTH1/PTH2 only).
func (v Value) IsAbsolute() bool { return v.IsPTH1() && v.TPIdent == TPIdentAbsolute }

// IsRelative reports whether the path is tagged as relative (PTH1/PTH2 only).
func (v Value) IsRelative() bool { return v.IsPTH1() && v.TPIdent == TPIdentRelative }

// IsUNC reports whether the path is tagged as a UNC path (PTH1/PTH2 only).
func (v Value) IsUNC() bool { return v.IsPTH1() && v.TPIdent == TPIdentUNC }

// IsNotAPath reports whether the path is tagged with the "!pth" placeholder
// LabVIEW uses for "no path here" (PTH1/PTH2 only).
func (v Value) IsNotAPath() bool { return v.IsPTH1() && v.TPIdent == TPIdentNotPath }

// IsPhony reports whether this is the PTH0 zero-fill phony form.
func (v Value) IsPhony() bool { return v.ZeroFill && v.IsPTH0() }

// Decode parses a path reference at the start of buf and returns the
// decoded Value plus the number of bytes it consumed. Callers that
// embed paths inside larger payloads (e.g. LIfp / LIbd entries) can use
// the returned byte count to advance their own cursor.
func Decode(buf []byte) (Value, int, error) {
	if len(buf) < 8 {
		return Value{}, 0, fmt.Errorf("pthx: payload too short for header: %d bytes (need at least 8)", len(buf))
	}
	ident := string(buf[:4])
	totlen := int(binary.BigEndian.Uint32(buf[4:8]))
	if 8+totlen > len(buf) {
		return Value{}, 0, fmt.Errorf("pthx: declared totlen %d exceeds available %d bytes", totlen, len(buf)-8)
	}

	switch ident {
	case IdentPTH0:
		return decodePTH0(ident, totlen, buf)
	case IdentPTH1, IdentPTH2:
		return decodePTH1(ident, totlen, buf)
	default:
		return Value{}, 0, fmt.Errorf("pthx: unknown path ident %q", ident)
	}
}

func decodePTH0(ident string, totlen int, buf []byte) (Value, int, error) {
	// LabVIEW's "phony" form: totlen == 0 means the path is just the
	// 8-byte envelope with no body.
	if totlen == 0 {
		return Value{Ident: ident, ZeroFill: true}, 8, nil
	}
	if totlen < 4 {
		return Value{}, 0, fmt.Errorf("pthx: PTH0 totlen %d too small (need 4 for tpval+count)", totlen)
	}
	body := buf[8 : 8+totlen]
	tpval := binary.BigEndian.Uint16(body[0:2])
	count := int(binary.BigEndian.Uint16(body[2:4]))
	pos := 4
	components := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		if pos >= len(body) {
			return Value{}, 0, fmt.Errorf("pthx: PTH0 truncated reading length of component %d", i)
		}
		n := int(body[pos])
		pos++
		if pos+n > len(body) {
			return Value{}, 0, fmt.Errorf("pthx: PTH0 component %d overruns body (need %d, have %d)", i, n, len(body)-pos)
		}
		comp := make([]byte, n)
		copy(comp, body[pos:pos+n])
		components = append(components, comp)
		pos += n
	}
	if pos != len(body) {
		return Value{}, 0, fmt.Errorf("pthx: PTH0 has %d trailing byte(s) after %d component(s)", len(body)-pos, count)
	}
	return Value{
		Ident:      ident,
		TPVal:      tpval,
		Components: components,
	}, 8 + totlen, nil
}

func decodePTH1(ident string, totlen int, buf []byte) (Value, int, error) {
	if totlen < 4 {
		return Value{}, 0, fmt.Errorf("pthx: %s totlen %d too small (need 4 for tpident)", ident, totlen)
	}
	body := buf[8 : 8+totlen]
	tpident := string(body[0:4])
	pos := 4
	var components [][]byte
	for pos < len(body) {
		if pos+2 > len(body) {
			return Value{}, 0, fmt.Errorf("pthx: %s truncated reading length of component %d", ident, len(components))
		}
		n := int(binary.BigEndian.Uint16(body[pos : pos+2]))
		pos += 2
		if pos+n > len(body) {
			return Value{}, 0, fmt.Errorf("pthx: %s component %d overruns body (need %d, have %d)", ident, len(components), n, len(body)-pos)
		}
		comp := make([]byte, n)
		copy(comp, body[pos:pos+n])
		components = append(components, comp)
		pos += n
	}
	return Value{
		Ident:      ident,
		TPIdent:    tpident,
		Components: components,
	}, 8 + totlen, nil
}

// Encode serializes a Value to its on-disk byte form.
func Encode(v Value) ([]byte, error) {
	switch v.Ident {
	case IdentPTH0:
		return encodePTH0(v)
	case IdentPTH1, IdentPTH2:
		return encodePTH1(v)
	default:
		return nil, fmt.Errorf("pthx: cannot encode unknown ident %q", v.Ident)
	}
}

func encodePTH0(v Value) ([]byte, error) {
	if v.ZeroFill {
		// 8-byte phony form: just the FourCC + four zero bytes.
		out := make([]byte, 8)
		copy(out, v.Ident)
		return out, nil
	}
	bodyLen := 4 // tpval + count
	for i, c := range v.Components {
		if len(c) > 0xFF {
			return nil, fmt.Errorf("pthx: PTH0 component %d is %d bytes, max 255", i, len(c))
		}
		bodyLen += 1 + len(c)
	}
	out := make([]byte, 8+bodyLen)
	copy(out[0:4], v.Ident)
	binary.BigEndian.PutUint32(out[4:8], uint32(bodyLen))
	binary.BigEndian.PutUint16(out[8:10], v.TPVal)
	binary.BigEndian.PutUint16(out[10:12], uint16(len(v.Components)))
	pos := 12
	for _, c := range v.Components {
		out[pos] = byte(len(c))
		pos++
		copy(out[pos:], c)
		pos += len(c)
	}
	return out, nil
}

func encodePTH1(v Value) ([]byte, error) {
	if len(v.TPIdent) != 4 {
		return nil, fmt.Errorf("pthx: %s requires 4-byte TPIdent, got %q", v.Ident, v.TPIdent)
	}
	bodyLen := 4 // tpident
	for i, c := range v.Components {
		if len(c) > 0xFFFF {
			return nil, fmt.Errorf("pthx: %s component %d is %d bytes, max 65535", v.Ident, i, len(c))
		}
		bodyLen += 2 + len(c)
	}
	out := make([]byte, 8+bodyLen)
	copy(out[0:4], v.Ident)
	binary.BigEndian.PutUint32(out[4:8], uint32(bodyLen))
	copy(out[8:12], v.TPIdent)
	pos := 12
	for _, c := range v.Components {
		binary.BigEndian.PutUint16(out[pos:pos+2], uint16(len(c)))
		pos += 2
		copy(out[pos:], c)
		pos += len(c)
	}
	return out, nil
}
