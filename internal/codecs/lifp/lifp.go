// Package lifp implements a narrow codec for the "LIfp" resource, which
// carries front-panel metadata/import references.
//
// This codec deliberately models only the stable outer structure observed in
// the committed corpus:
//   - u16 version
//   - 4-byte marker "FPHP"
//   - u32 entry count
//   - repeated entries with Pascal-string qualifiers and raw PTH0 path refs
//   - u16 footer
//
// Embedded path references are preserved byte-for-byte via PathRef.Raw so the
// decoded form can be re-encoded exactly even though the inner PTH0 structure
// is not interpreted semantically yet.
package lifp

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

const (
	// FourCC is the resource type this codec handles.
	FourCC codecs.FourCC = "LIfp"

	headerSize = 10
	footerSize = 2
	entryTail  = 16

	expectedMarker = "FPHP"
)

// Value is the decoded form of an LIfp payload.
type Value struct {
	Version    uint16
	Marker     string
	EntryCount uint32
	Entries    []Entry
	Footer     uint16
}

// Entry is one front-panel metadata/import reference.
type Entry struct {
	Kind           uint16
	LinkType       string
	QualifierCount uint32
	Qualifiers     []string
	PrimaryPath    PathRef
	Field0         uint32
	Field1         uint32
	Field2         uint32
	Field3         uint32
	SecondaryPath  *PathRef

	qualifierPad []byte
}

// PathRef preserves an embedded path reference exactly as found on disk.
type PathRef struct {
	Class       string
	DeclaredLen uint32
	Raw         []byte
}

type parseKey struct {
	suffixLen int
	remaining int
}

type parseResult struct {
	entries  []Entry
	footer   uint16
	consumed int
	err      error
	recorded bool
}

// Codec implements codecs.ResourceCodec for LIfp.
type Codec struct{}

// Capability reports the codec's static metadata.
func (Codec) Capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        FourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier1,
	}
}

// Decode parses payload into a Value.
func (Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	return decodeValue(payload, true)
}

// Encode serializes a Value (by value or pointer) into the observed wire
// format. Unknown PTH0 internals are emitted from PathRef.Raw unchanged.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	v, err := coerceValue(value)
	if err != nil {
		return nil, err
	}
	if v.Marker != expectedMarker {
		return nil, fmt.Errorf("LIfp: Marker = %q, want %q", v.Marker, expectedMarker)
	}
	if v.EntryCount != uint32(len(v.Entries)) {
		return nil, fmt.Errorf("LIfp: EntryCount = %d, want %d entries", v.EntryCount, len(v.Entries))
	}

	out := make([]byte, 0, headerSize+footerSize)
	out = appendU16(out, v.Version)
	out = append(out, v.Marker...)
	out = appendU32(out, v.EntryCount)
	for i, entry := range v.Entries {
		if entry.QualifierCount != uint32(len(entry.Qualifiers)) {
			return nil, fmt.Errorf("LIfp: entry %d QualifierCount = %d, want %d qualifiers", i, entry.QualifierCount, len(entry.Qualifiers))
		}
		out = appendU16(out, entry.Kind)
		if len(entry.LinkType) != 4 {
			return nil, fmt.Errorf("LIfp: entry %d LinkType length = %d, want 4", i, len(entry.LinkType))
		}
		out = append(out, entry.LinkType...)
		out = appendU32(out, entry.QualifierCount)
		for j, qualifier := range entry.Qualifiers {
			if len(qualifier) > 0xff {
				return nil, fmt.Errorf("LIfp: entry %d qualifier %d length = %d, exceeds 255", i, j, len(qualifier))
			}
			out = append(out, byte(len(qualifier)))
			out = append(out, qualifier...)
		}
		out = append(out, entry.qualifierPad...)
		pathBytes, err := encodePathRef(entry.PrimaryPath, fmt.Sprintf("entry %d primary path", i))
		if err != nil {
			return nil, err
		}
		out = append(out, pathBytes...)
		out = appendU32(out, entry.Field0)
		out = appendU32(out, entry.Field1)
		out = appendU32(out, entry.Field2)
		out = appendU32(out, entry.Field3)
		if entry.SecondaryPath != nil {
			pathBytes, err := encodePathRef(*entry.SecondaryPath, fmt.Sprintf("entry %d secondary path", i))
			if err != nil {
				return nil, err
			}
			out = append(out, pathBytes...)
		}
	}
	out = appendU16(out, v.Footer)
	return out, nil
}

// Validate reports structural issues with payload.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	loc := validate.IssueLocation{Area: "LIfp", BlockType: string(FourCC)}
	if len(payload) < headerSize+footerSize {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "lifp.payload.short",
			Message:  fmt.Sprintf("LIfp payload is %d bytes, need at least %d", len(payload), headerSize+footerSize),
			Location: loc,
		}}
	}
	if string(payload[2:6]) != expectedMarker {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "lifp.marker.invalid",
			Message:  fmt.Sprintf("LIfp marker = %q, want %q", string(payload[2:6]), expectedMarker),
			Location: loc,
		}}
	}
	if _, err := decodeValue(payload, true); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "lifp.decode.invalid",
			Message:  err.Error(),
			Location: loc,
		}}
	}
	return nil
}

func coerceValue(value any) (Value, error) {
	switch tv := value.(type) {
	case Value:
		return tv, nil
	case *Value:
		if tv == nil {
			return Value{}, fmt.Errorf("LIfp: Encode received nil *Value")
		}
		return *tv, nil
	default:
		return Value{}, fmt.Errorf("LIfp: Encode expected Value or *Value, got %T", value)
	}
}

func decodeValue(payload []byte, strictMarker bool) (Value, error) {
	if len(payload) < headerSize+footerSize {
		return Value{}, fmt.Errorf("LIfp: payload too short: %d bytes (need at least %d)", len(payload), headerSize+footerSize)
	}

	v := Value{
		Version:    binary.BigEndian.Uint16(payload[:2]),
		Marker:     string(payload[2:6]),
		EntryCount: binary.BigEndian.Uint32(payload[6:10]),
	}
	if strictMarker && v.Marker != expectedMarker {
		return Value{}, fmt.Errorf("LIfp: marker = %q, want %q", v.Marker, expectedMarker)
	}

	rest := payload[headerSize:]
	memo := make(map[parseKey]parseResult)
	entries, footer, consumed, err := decodeEntries(rest, int(v.EntryCount), memo)
	if err != nil {
		return Value{}, err
	}
	if consumed != len(rest) {
		return Value{}, fmt.Errorf("LIfp: trailing payload size = %d", len(rest)-consumed)
	}
	v.Entries = entries
	v.Footer = footer
	return v, nil
}

func decodeEntries(payload []byte, remaining int, memo map[parseKey]parseResult) ([]Entry, uint16, int, error) {
	key := parseKey{suffixLen: len(payload), remaining: remaining}
	if cached, ok := memo[key]; ok && cached.recorded {
		return cached.entries, cached.footer, cached.consumed, cached.err
	}

	var result parseResult
	if remaining == 0 {
		if len(payload) != footerSize {
			result = parseResult{
				err:      fmt.Errorf("LIfp: footer size = %d, want %d", len(payload), footerSize),
				recorded: true,
			}
			memo[key] = result
			return nil, 0, 0, result.err
		}
		result = parseResult{
			footer:   binary.BigEndian.Uint16(payload[:footerSize]),
			consumed: footerSize,
			recorded: true,
		}
		memo[key] = result
		return nil, result.footer, result.consumed, nil
	}

	entry, consumed, err := decodeEntry(payload, remaining, memo)
	if err != nil {
		result = parseResult{err: err, recorded: true}
		memo[key] = result
		return nil, 0, 0, err
	}
	nextEntries, footer, nextConsumed, err := decodeEntries(payload[consumed:], remaining-1, memo)
	if err != nil {
		result = parseResult{err: err, recorded: true}
		memo[key] = result
		return nil, 0, 0, err
	}
	result = parseResult{
		entries:  append([]Entry{entry}, nextEntries...),
		footer:   footer,
		consumed: consumed + nextConsumed,
		recorded: true,
	}
	memo[key] = result
	return result.entries, result.footer, result.consumed, nil
}

func decodeEntry(payload []byte, remaining int, memo map[parseKey]parseResult) (Entry, int, error) {
	if len(payload) < 10 {
		return Entry{}, 0, fmt.Errorf("entry truncated before qualifier list")
	}
	consumed := 0
	e := Entry{
		Kind:           binary.BigEndian.Uint16(payload[consumed : consumed+2]),
		LinkType:       string(payload[consumed+2 : consumed+6]),
		QualifierCount: binary.BigEndian.Uint32(payload[consumed+6 : consumed+10]),
	}
	consumed += 10

	e.Qualifiers = make([]string, 0, e.QualifierCount)
	for i := uint32(0); i < e.QualifierCount; i++ {
		s, n, err := decodePascalString(payload[consumed:])
		if err != nil {
			return Entry{}, 0, fmt.Errorf("qualifier %d: %w", i, err)
		}
		e.Qualifiers = append(e.Qualifiers, s)
		consumed += n
	}

	pathStart, pad, err := locatePrimaryPathStart(payload, consumed)
	if err != nil {
		return Entry{}, 0, err
	}
	if len(pad) != 0 {
		e.qualifierPad = append([]byte(nil), pad...)
	}

	for _, primaryPath := range candidatePathRefs(payload[pathStart:]) {
		e.SecondaryPath = nil
		tailStart := pathStart + len(primaryPath.Raw)
		if len(payload[tailStart:]) < entryTail {
			continue
		}
		e.PrimaryPath = primaryPath
		e.Field0 = binary.BigEndian.Uint32(payload[tailStart : tailStart+4])
		e.Field1 = binary.BigEndian.Uint32(payload[tailStart+4 : tailStart+8])
		e.Field2 = binary.BigEndian.Uint32(payload[tailStart+8 : tailStart+12])
		e.Field3 = binary.BigEndian.Uint32(payload[tailStart+12 : tailStart+16])
		afterTail := tailStart + entryTail

		if _, _, _, err := decodeEntries(payload[afterTail:], remaining-1, memo); err == nil {
			return e, afterTail, nil
		}
		if len(payload[afterTail:]) < 4 || string(payload[afterTail:afterTail+4]) != "PTH0" {
			continue
		}
		for _, secondaryPath := range candidatePathRefs(payload[afterTail:]) {
			secondaryEnd := afterTail + len(secondaryPath.Raw)
			if _, _, _, err := decodeEntries(payload[secondaryEnd:], remaining-1, memo); err == nil {
				e.SecondaryPath = &secondaryPath
				return e, secondaryEnd, nil
			}
		}
	}

	return Entry{}, 0, fmt.Errorf("could not locate path/tail boundaries")
}

func decodePascalString(payload []byte) (string, int, error) {
	if len(payload) == 0 {
		return "", 0, fmt.Errorf("string length byte missing")
	}
	size := int(payload[0])
	if len(payload) < 1+size {
		return "", 0, fmt.Errorf("string length %d overruns remaining %d bytes", size, len(payload)-1)
	}
	return string(payload[1 : 1+size]), 1 + size, nil
}

func locatePrimaryPathStart(payload []byte, start int) (int, []byte, error) {
	for padLen := 0; padLen <= 3; padLen++ {
		pathStart := start + padLen
		if pathStart+4 > len(payload) {
			break
		}
		if string(payload[pathStart:pathStart+4]) != "PTH0" {
			continue
		}
		pad := payload[start:pathStart]
		allZero := true
		for _, b := range pad {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			return pathStart, pad, nil
		}
	}
	return 0, nil, fmt.Errorf("primary path marker missing after qualifiers")
}

func encodePathRef(path PathRef, label string) ([]byte, error) {
	if len(path.Class) != 4 {
		return nil, fmt.Errorf("LIfp: %s class length = %d, want 4", label, len(path.Class))
	}
	if len(path.Raw) == 0 {
		raw := make([]byte, 8+int(path.DeclaredLen))
		copy(raw[:4], path.Class)
		binary.BigEndian.PutUint32(raw[4:8], path.DeclaredLen)
		return raw, nil
	}
	if string(path.Raw[:4]) != path.Class {
		return nil, fmt.Errorf("LIfp: %s raw class = %q, want %q", label, string(path.Raw[:4]), path.Class)
	}
	if binary.BigEndian.Uint32(path.Raw[4:8]) != path.DeclaredLen {
		return nil, fmt.Errorf("LIfp: %s raw declared length = %d, want %d", label, binary.BigEndian.Uint32(path.Raw[4:8]), path.DeclaredLen)
	}
	raw := make([]byte, len(path.Raw))
	copy(raw, path.Raw)
	return raw, nil
}

func candidatePathRefs(payload []byte) []PathRef {
	if len(payload) < 8 || string(payload[:4]) != "PTH0" {
		return nil
	}
	declaredLen := int(binary.BigEndian.Uint32(payload[4:8]))
	base := 8 + declaredLen
	if base > len(payload) {
		return nil
	}
	out := make([]PathRef, 0, 5)
	for extra := 0; extra <= 4; extra++ {
		total := base + extra
		if total > len(payload) {
			break
		}
		if !allZero(payload[base:total]) {
			continue
		}
		raw := make([]byte, total)
		copy(raw, payload[:total])
		out = append(out, PathRef{
			Class:       "PTH0",
			DeclaredLen: uint32(declaredLen),
			Raw:         raw,
		})
	}
	return out
}

func appendU16(dst []byte, v uint16) []byte {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	return append(dst, buf[:]...)
}

func appendU32(dst []byte, v uint32) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	return append(dst, buf[:]...)
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
