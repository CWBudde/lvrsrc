package lvrsrc_test

import (
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestParseLenientByDefaultForRecoverableSecondaryHeaderMismatch(t *testing.T) {
	data := readFixture(t, "config-data.ctl")
	infoOffset := int(readU32BE(t, data, 16))
	data[infoOffset+8] ^= 0x01

	if _, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
}

func TestParseStrictRejectsRecoverableSecondaryHeaderMismatch(t *testing.T) {
	data := readFixture(t, "config-data.ctl")
	infoOffset := int(readU32BE(t, data, 16))
	data[infoOffset+8] ^= 0x01

	if _, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{Strict: true}); err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}
}

func readU32BE(t *testing.T, data []byte, off int) uint32 {
	t.Helper()
	if off+4 > len(data) {
		t.Fatalf("readU32BE(%d) out of bounds for len=%d", off, len(data))
	}

	return uint32(data[off])<<24 |
		uint32(data[off+1])<<16 |
		uint32(data[off+2])<<8 |
		uint32(data[off+3])
}
