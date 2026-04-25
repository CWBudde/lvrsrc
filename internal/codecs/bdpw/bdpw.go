// Package bdpw implements the codec for the "BDPW" resource — Block
// Diagram Password. The payload is three concatenated 16-byte MD5 hashes:
// the password digest, plus two derived "verification" hashes LabVIEW
// recomputes on save.
//
// Empty-password sentinel: an unprotected VI carries
// `d41d8cd98f00b204e9800998ecf8427e` as `PasswordMD5`, the MD5 of the
// empty string.
//
// References:
//   - pylabview LVblock.py:4334-4680 (BDPW class with hash_1 and hash_2
//     derivation logic).
//   - pylavi resource_types.py:54-94 (TypeBDPW with the empty-password
//     sentinel and a `password_matches` helper).
//
// Older LabVIEW versions (< 8.0) ship BDPW without `Hash2` (32-byte
// payload). The corpus contains only the modern 48-byte form; the codec
// accepts only that variant for now and rejects others as malformed.
//
// Safety tier: 1 (read-only). The codec deliberately does not support
// writing a non-zero password — `Hash1` and `Hash2` are derived from the
// VI's contents and we have not validated the derivation against
// arbitrary VIs.
//
// See docs/resources/bdpw.md for layout, semantics, and open questions.
package bdpw

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "BDPW"

// hashSize is the byte length of one MD5 hash.
const hashSize = 16

// payloadSize is the canonical BDPW payload size for LV >= 8.0
// (PasswordMD5 || Hash1 || Hash2). Older LabVIEW versions omit Hash2 for
// a 32-byte payload, but the shipped corpus is uniformly 48 bytes.
const payloadSize = 3 * hashSize

// emptyPasswordMD5 is the MD5 of the empty string. A BDPW with this as
// the first hash means the VI is unprotected.
var emptyPasswordMD5 = md5Sum("")

// Value is the decoded form of a BDPW payload.
type Value struct {
	// PasswordMD5 is the MD5 hash of the password. When the VI is
	// unprotected this matches md5("") (see HasPassword).
	PasswordMD5 [hashSize]byte
	// Hash1 is LabVIEW's first derived verification hash.
	Hash1 [hashSize]byte
	// Hash2 is LabVIEW's second derived verification hash (added in
	// LV 8.0; absent in older formats).
	Hash2 [hashSize]byte
}

// HasPassword returns true when the password hash is not the
// empty-password sentinel. It does not attempt to crack the hash.
func (v Value) HasPassword() bool {
	return v.PasswordMD5 != emptyPasswordMD5
}

// PasswordMatches checks whether the given password's MD5 matches
// PasswordMD5. Verifying a candidate password is safe (no derivation
// secrets are stored client-side); changing the password is not
// supported by this codec.
func (v Value) PasswordMatches(password string) bool {
	got := md5.Sum([]byte(password))
	return got == v.PasswordMD5
}

// EmptyPasswordHex returns the hex-encoded empty-password MD5 sentinel
// for callers that want to display or compare it.
func EmptyPasswordHex() string {
	return hex.EncodeToString(emptyPasswordMD5[:])
}

// Codec implements codecs.ResourceCodec for the BDPW resource.
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
	if len(payload) != payloadSize {
		return nil, fmt.Errorf("BDPW: payload size = %d, want %d (LV>=8.0 with Hash2)", len(payload), payloadSize)
	}
	var v Value
	copy(v.PasswordMD5[:], payload[0:hashSize])
	copy(v.Hash1[:], payload[hashSize:2*hashSize])
	copy(v.Hash2[:], payload[2*hashSize:3*hashSize])
	return v, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("BDPW: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("BDPW: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, payloadSize)
	copy(out[0:hashSize], v.PasswordMD5[:])
	copy(out[hashSize:2*hashSize], v.Hash1[:])
	copy(out[2*hashSize:3*hashSize], v.Hash2[:])
	return out, nil
}

// Validate reports structural issues.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "bdpw.payload.size",
		Message:  fmt.Sprintf("BDPW payload is %d bytes, want %d", len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}

// md5Sum returns the MD5 of s as a [16]byte for compile-time use in
// emptyPasswordMD5.
func md5Sum(s string) [hashSize]byte {
	return md5.Sum([]byte(s))
}
