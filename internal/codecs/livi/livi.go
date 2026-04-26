// Package livi implements a codec for the "LIvi" resource — the
// VI-level link-info block. It records dependencies between this VI/CTL
// and other VIs, classes, and libraries.
//
// The on-disk layout mirrors LIfp / LIbd: an outer envelope (start
// marker + file-kind ident + entry count) wraps a list of typed link
// references followed by an end marker. Each entry has a 2-byte
// continuation marker, a 4-byte LinkObjRef ident, a qualifier list, an
// optional alignment pad, the primary PTH0 path, an opaque "tail" of
// post-path bytes (which Entry.Target() decodes through internal/codecs/
// linkobj), and an optional secondary PTH0 path (the viLSPathRef
// captured by HeapToVILinkSaveInfo).
//
// pylabview reference: LVblock.py:2248 (LinkObjRefs base) and
// LVblock.py:2426 (LIvi). The 4-byte ident slot was previously called
// "Marker" in this codec; the underlying field is the section.ident
// from pylabview, which is what discriminates LVIN (.vi), LVCC (.ctl),
// LVIT (.vit), and LLBV (.llb) link lists.
package livi

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/linkobj"
	"github.com/CWBudde/lvrsrc/internal/codecs/pthx"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

const (
	// FourCC is the resource type this codec handles.
	FourCC codecs.FourCC = "LIvi"

	headerSize = 10 // u16 startMarker + 4-byte ident + u32 entry count
	footerSize = 2  // u16 endMarker
)

// knownMarkers enumerates the file-kind FourCCs LIvi is observed to
// carry. Decode accepts any 4-byte marker (Encode preserves whatever the
// caller supplies); Validate flags unknown markers as a warning so the
// corpus surface stays documented.
var knownMarkers = map[string]struct{}{
	"LVIN": {}, // VI
	"LVCC": {}, // CTL (control)
	"LVIT": {}, // VIT (template)
	"LLBV": {}, // LLB (library)
}

func isKnownMarker(s string) bool {
	_, ok := knownMarkers[s]
	return ok
}

// Value is the decoded form of an LIvi payload.
type Value struct {
	// Version is the u16 BE word at offset 0. pylabview reads this as the
	// "nextLinkInfo=1" list-start marker; we keep the legacy field name
	// for backwards compatibility but it always equals 1 in valid files.
	Version uint16
	// Marker is the 4-byte file-kind FourCC at offset 2 (LVIN, LVCC,
	// LVIT, LLBV).
	Marker string
	// EntryCount is the u32 BE count at offset 6.
	EntryCount uint32
	// Entries lists the typed link references in on-disk order.
	Entries []Entry
	// Footer is the u16 BE word at the tail of the payload. pylabview
	// expects this to be the "nextLinkInfo=3" list-end marker (= 3).
	Footer uint16
}

// Entry is one VI-level link reference. Its shape matches LIfp / LIbd
// entries; see those packages for additional documentation.
type Entry struct {
	Kind           uint16
	LinkType       string
	QualifierCount uint32
	Qualifiers     []string
	PrimaryPath    PathRef
	Tail           []byte
	SecondaryPath  *PathRef

	// prefixPad is the 0..3 byte pre-qualifier alignment pad pylabview's
	// parseBasicLinkSaveInfo inserts to keep the qualified-name list on
	// a 4-byte boundary. LIfp / LIbd entries in the shipped corpus
	// happen to be aligned without this pad; LIvi entries hit it
	// whenever the previous entry's encoded size is not a multiple of
	// four (e.g. a 39-byte VILB followed by a VICC).
	prefixPad []byte

	// qualifierPad is the 0..3 byte post-qualifier alignment pad before
	// the primary PTH0 path (the 2-byte align in
	// parseBasicLinkSaveInfo).
	qualifierPad []byte
}

// PathRef preserves an embedded path reference exactly as found on disk.
type PathRef struct {
	Class       string
	DeclaredLen uint32
	Raw         []byte
}

// Decoded parses the path through internal/codecs/pthx and returns the
// typed Value. Errors indicate the on-disk bytes did not match a PTH0 /
// PTH1 / PTH2 layout — round-trip preservation is unaffected because
// callers should still use Raw for re-encoding.
func (p PathRef) Decoded() (pthx.Value, error) {
	v, _, err := pthx.Decode(p.Raw)
	return v, err
}

// Target decodes the entry's post-primary-path payload into a typed
// linkobj.LinkTarget. The decode is lazy: round-trip serialization
// continues to use Tail and SecondaryPath as the byte-authoritative
// source.
func (e Entry) Target() (linkobj.LinkTarget, error) {
	var secondaryRaw []byte
	if e.SecondaryPath != nil {
		secondaryRaw = e.SecondaryPath.Raw
	}
	return linkobj.Decode(e.LinkType, e.Tail, secondaryRaw)
}

// Codec implements codecs.ResourceCodec for the LIvi resource.
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
	return decodeValue(payload)
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	v, err := coerceValue(value)
	if err != nil {
		return nil, err
	}
	if len(v.Marker) != 4 {
		return nil, fmt.Errorf("LIvi: Marker length = %d, want 4", len(v.Marker))
	}
	if v.EntryCount != uint32(len(v.Entries)) {
		return nil, fmt.Errorf("LIvi: EntryCount = %d, want %d entries", v.EntryCount, len(v.Entries))
	}

	out := make([]byte, 0, headerSize+footerSize)
	out = appendU16(out, v.Version)
	out = append(out, v.Marker...)
	out = appendU32(out, v.EntryCount)
	for i, entry := range v.Entries {
		if entry.QualifierCount != uint32(len(entry.Qualifiers)) {
			return nil, fmt.Errorf("LIvi: entry %d QualifierCount = %d, want %d qualifiers", i, entry.QualifierCount, len(entry.Qualifiers))
		}
		out = appendU16(out, entry.Kind)
		if len(entry.LinkType) != 4 {
			return nil, fmt.Errorf("LIvi: entry %d LinkType length = %d, want 4", i, len(entry.LinkType))
		}
		out = append(out, entry.LinkType...)
		out = append(out, entry.prefixPad...)
		out = appendU32(out, entry.QualifierCount)
		for j, qualifier := range entry.Qualifiers {
			if len(qualifier) > 0xff {
				return nil, fmt.Errorf("LIvi: entry %d qualifier %d length = %d, exceeds 255", i, j, len(qualifier))
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
		out = append(out, entry.Tail...)
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
	loc := validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)}
	if len(payload) < headerSize+footerSize {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "livi.payload.short",
			Message:  fmt.Sprintf("LIvi payload is %d bytes, need at least %d", len(payload), headerSize+footerSize),
			Location: loc,
		}}
	}
	marker := string(payload[2:6])
	if !isKnownMarker(marker) {
		return []validate.Issue{{
			Severity: validate.SeverityWarning,
			Code:     "livi.marker.unknown",
			Message:  fmt.Sprintf("LIvi marker = %q is not in the known set (LVIN/LVCC/LVIT/LLBV)", marker),
			Location: loc,
		}}
	}
	if _, err := decodeValue(payload); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "livi.decode.invalid",
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
			return Value{}, fmt.Errorf("LIvi: Encode received nil *Value")
		}
		return *tv, nil
	default:
		return Value{}, fmt.Errorf("LIvi: Encode expected Value or *Value, got %T", value)
	}
}

func decodeValue(payload []byte) (Value, error) {
	if len(payload) < headerSize+footerSize {
		return Value{}, fmt.Errorf("LIvi: payload too short: %d bytes (need at least %d)", len(payload), headerSize+footerSize)
	}
	v := Value{
		Version:    binary.BigEndian.Uint16(payload[0:2]),
		Marker:     string(payload[2:6]),
		EntryCount: binary.BigEndian.Uint32(payload[6:10]),
	}
	rest := payload[headerSize:]
	memo := make(map[parseKey]parseResult)
	entries, footer, consumed, err := decodeEntries(rest, int(v.EntryCount), memo)
	if err != nil {
		return Value{}, err
	}
	if consumed != len(rest) {
		return Value{}, fmt.Errorf("LIvi: trailing payload size = %d", len(rest)-consumed)
	}
	v.Entries = entries
	v.Footer = footer
	return v, nil
}

// The entry-parser below is structurally identical to the one in lifp /
// libd. The complexity comes from the boundary heuristic — qualifier
// padding is 0..3 bytes, primary PTH0 size has 0..4 trailing zero
// candidates, and the secondary PTH0 (if present) extends the entry —
// so we try every combination and memoise per (suffix length, entries
// remaining) to keep parsing linear. See lifp.go for the original.

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

func decodeEntries(payload []byte, remaining int, memo map[parseKey]parseResult) ([]Entry, uint16, int, error) {
	key := parseKey{suffixLen: len(payload), remaining: remaining}
	if cached, ok := memo[key]; ok && cached.recorded {
		return cached.entries, cached.footer, cached.consumed, cached.err
	}
	var result parseResult
	if remaining == 0 {
		if len(payload) != footerSize {
			result = parseResult{err: fmt.Errorf("LIvi: footer size = %d, want %d", len(payload), footerSize), recorded: true}
			memo[key] = result
			return nil, 0, 0, result.err
		}
		result = parseResult{footer: binary.BigEndian.Uint16(payload[:footerSize]), consumed: footerSize, recorded: true}
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
	// pylabview's parseBasicLinkSaveInfo aligns to 4 bytes before
	// reading the qualified-name count. The pad lives in the section
	// stream between the LinkObj's ident (4 bytes after the
	// nextLinkInfo word) and the qualified-name count. Since we don't
	// know how many bytes were consumed before this entry — and
	// callers don't tell us — we try every plausible width (0..3) and
	// keep the first one where the qualifier count parses sanely.
	for prefix := 0; prefix <= 3; prefix++ {
		if 6+prefix+4 > len(payload) {
			break
		}
		// The pad bytes are always zeros in the corpus.
		if !allZero(payload[6 : 6+prefix]) {
			continue
		}
		e, used, ok := tryDecodeEntry(payload, prefix, remaining, memo)
		if ok {
			return e, used, nil
		}
	}
	return Entry{}, 0, fmt.Errorf("could not locate path/tail boundaries")
}

func tryDecodeEntry(payload []byte, prefixPad, remaining int, memo map[parseKey]parseResult) (Entry, int, bool) {
	consumed := 0
	e := Entry{
		Kind:     binary.BigEndian.Uint16(payload[consumed : consumed+2]),
		LinkType: string(payload[consumed+2 : consumed+6]),
	}
	consumed += 6
	if prefixPad > 0 {
		e.prefixPad = append([]byte(nil), payload[consumed:consumed+prefixPad]...)
	}
	consumed += prefixPad
	if consumed+4 > len(payload) {
		return Entry{}, 0, false
	}
	qcount := binary.BigEndian.Uint32(payload[consumed : consumed+4])
	const maxQuals = 64 // sanity cap; corpus seen up to 5 qualifiers
	if qcount > maxQuals {
		return Entry{}, 0, false
	}
	e.QualifierCount = qcount
	consumed += 4

	e.Qualifiers = make([]string, 0, e.QualifierCount)
	for i := uint32(0); i < e.QualifierCount; i++ {
		if consumed >= len(payload) {
			return Entry{}, 0, false
		}
		s, n, err := decodePascalString(payload[consumed:])
		if err != nil {
			return Entry{}, 0, false
		}
		e.Qualifiers = append(e.Qualifiers, s)
		consumed += n
	}

	pathStart, pad, err := locatePrimaryPathStart(payload, consumed)
	if err != nil {
		return Entry{}, 0, false
	}
	if len(pad) != 0 {
		e.qualifierPad = append([]byte(nil), pad...)
	}

	for _, primaryPath := range candidatePathRefs(payload[pathStart:]) {
		e.SecondaryPath = nil
		tailStart := pathStart + len(primaryPath.Raw)
		e.PrimaryPath = primaryPath
		if remaining == 1 {
			entryEnd := len(payload) - footerSize
			if entryEnd < tailStart {
				continue
			}
			tail, secondary, ok := splitTailAndSecondary(payload[tailStart:entryEnd])
			if !ok {
				continue
			}
			e.Tail = tail
			e.SecondaryPath = secondary
			return e, entryEnd, true
		}
		for nextStart := tailStart; nextStart+10 <= len(payload)-footerSize; nextStart++ {
			if !looksLikeEntryHeader(payload[nextStart:]) {
				continue
			}
			if _, _, _, err := decodeEntries(payload[nextStart:], remaining-1, memo); err == nil {
				tail, secondary, ok := splitTailAndSecondary(payload[tailStart:nextStart])
				if !ok {
					continue
				}
				e.Tail = tail
				e.SecondaryPath = secondary
				return e, nextStart, true
			}
		}
	}
	return Entry{}, 0, false
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
		return nil, fmt.Errorf("LIvi: %s class length = %d, want 4", label, len(path.Class))
	}
	if len(path.Raw) == 0 {
		raw := make([]byte, 8+int(path.DeclaredLen))
		copy(raw[:4], path.Class)
		binary.BigEndian.PutUint32(raw[4:8], path.DeclaredLen)
		return raw, nil
	}
	if string(path.Raw[:4]) != path.Class {
		return nil, fmt.Errorf("LIvi: %s raw class = %q, want %q", label, string(path.Raw[:4]), path.Class)
	}
	if binary.BigEndian.Uint32(path.Raw[4:8]) != path.DeclaredLen {
		return nil, fmt.Errorf("LIvi: %s raw declared length = %d, want %d", label, binary.BigEndian.Uint32(path.Raw[4:8]), path.DeclaredLen)
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
		out = append(out, PathRef{Class: "PTH0", DeclaredLen: uint32(declaredLen), Raw: raw})
	}
	return out
}

func splitTailAndSecondary(payload []byte) ([]byte, *PathRef, bool) {
	for i := 0; i+4 <= len(payload); i++ {
		if string(payload[i:i+4]) != "PTH0" {
			continue
		}
		for _, path := range candidatePathRefs(payload[i:]) {
			if len(path.Raw) == len(payload)-i {
				tail := make([]byte, i)
				copy(tail, payload[:i])
				pathCopy := path
				return tail, &pathCopy, true
			}
		}
	}
	tail := make([]byte, len(payload))
	copy(tail, payload)
	return tail, nil, true
}

func looksLikeEntryHeader(payload []byte) bool {
	if len(payload) < 10 {
		return false
	}
	if payload[0] != 0 {
		return false
	}
	for _, b := range payload[2:6] {
		if (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && (b < '0' || b > '9') {
			return false
		}
	}
	q := binary.BigEndian.Uint32(payload[6:10])
	return q <= 8
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
