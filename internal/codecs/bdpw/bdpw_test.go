package bdpw

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestCapability(t *testing.T) {
	c := Codec{}.Capability()
	if c.FourCC != FourCC {
		t.Fatalf("FourCC = %q, want %q", c.FourCC, FourCC)
	}
	if c.Safety != codecs.SafetyTier1 {
		t.Fatalf("Safety = %v, want SafetyTier1", c.Safety)
	}
}

func TestEmptyPasswordHexMatchesKnownConstant(t *testing.T) {
	if got, want := EmptyPasswordHex(), "d41d8cd98f00b204e9800998ecf8427e"; got != want {
		t.Errorf("EmptyPasswordHex() = %q, want %q", got, want)
	}
}

func makePayload(pwHash, h1, h2 []byte) []byte {
	out := make([]byte, payloadSize)
	copy(out[0:16], pwHash)
	copy(out[16:32], h1)
	copy(out[32:48], h2)
	return out
}

func TestDecodeExtractsThreeHashes(t *testing.T) {
	pw := bytesFromHex(t, "d41d8cd98f00b204e9800998ecf8427e") // empty
	h1 := bytesRepeat(0xAA, 16)
	h2 := bytesRepeat(0x55, 16)
	v, err := Codec{}.Decode(codecs.Context{}, makePayload(pw, h1, h2))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if !bytes.Equal(val.PasswordMD5[:], pw) {
		t.Errorf("PasswordMD5 = %x, want %x", val.PasswordMD5[:], pw)
	}
	if !bytes.Equal(val.Hash1[:], h1) {
		t.Errorf("Hash1 = %x, want %x", val.Hash1[:], h1)
	}
	if !bytes.Equal(val.Hash2[:], h2) {
		t.Errorf("Hash2 = %x, want %x", val.Hash2[:], h2)
	}
}

func TestDecodeRejectsWrongSize(t *testing.T) {
	for _, size := range []int{0, 32, 47, 49, 64} {
		if _, err := (Codec{}).Decode(codecs.Context{}, make([]byte, size)); err == nil {
			t.Errorf("Decode(%d-byte payload) returned nil error", size)
		}
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	original := makePayload(
		bytesRepeat(0x01, 16),
		bytesRepeat(0x23, 16),
		bytesRepeat(0x45, 16),
	)
	v, err := Codec{}.Decode(codecs.Context{}, original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original")
	}
}

func TestHasPasswordFalseForEmptySentinel(t *testing.T) {
	pw := bytesFromHex(t, "d41d8cd98f00b204e9800998ecf8427e")
	v, _ := Codec{}.Decode(codecs.Context{}, makePayload(pw, make([]byte, 16), make([]byte, 16)))
	if v.(Value).HasPassword() {
		t.Error("HasPassword() = true on empty-password sentinel")
	}
}

func TestHasPasswordTrueForNonEmptyHash(t *testing.T) {
	pw := bytesRepeat(0x12, 16) // not the sentinel
	v, _ := Codec{}.Decode(codecs.Context{}, makePayload(pw, make([]byte, 16), make([]byte, 16)))
	if !v.(Value).HasPassword() {
		t.Error("HasPassword() = false on non-empty hash")
	}
}

func TestPasswordMatchesAcceptsKnownPassword(t *testing.T) {
	const password = "secret"
	hash := md5.Sum([]byte(password))
	v, _ := Codec{}.Decode(codecs.Context{}, makePayload(hash[:], make([]byte, 16), make([]byte, 16)))
	if !v.(Value).PasswordMatches(password) {
		t.Error("PasswordMatches(\"secret\") = false")
	}
	if v.(Value).PasswordMatches("wrong") {
		t.Error("PasswordMatches(\"wrong\") = true")
	}
}

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, make([]byte, payloadSize)); len(issues) != 0 {
		t.Errorf("Validate(48 bytes) issues = %+v, want none", issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, make([]byte, 32))
	if len(issues) == 0 {
		t.Fatal("Validate(32 bytes) returned no issues")
	}
	if issues[0].Severity != validate.SeverityError {
		t.Errorf("severity = %v, want error", issues[0].Severity)
	}
}

func TestCorpusRoundTrip(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	total := 0
	withPassword := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("open %s: %v", e.Name(), err)
		}
		for _, block := range f.Blocks {
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				total++
				v, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("%s id=%d Decode: %v", e.Name(), section.Index, err)
				}
				if v.(Value).HasPassword() {
					withPassword++
				}
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("%s id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s id=%d round-trip mismatch", e.Name(), section.Index)
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no BDPW sections in corpus")
	}
	t.Logf("exercised %d BDPW section(s); %d had a non-empty password hash", total, withPassword)
}

func bytesFromHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex decode %q: %v", s, err)
	}
	return b
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
