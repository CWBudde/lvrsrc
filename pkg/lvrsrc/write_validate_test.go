package lvrsrc_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/lvrsrc/pkg/lvrsrc"
)

func TestWriteToRoundTrip(t *testing.T) {
	data := readFixture(t, "config-data.ctl")

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var buf bytes.Buffer
	if err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}

	roundTrip, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{Strict: true})
	if err != nil {
		t.Fatalf("Parse(WriteTo()) error = %v", err)
	}

	assertEquivalentFile(t, roundTrip, f)
}

func TestWriteToFileRoundTrip(t *testing.T) {
	data := readFixture(t, "config-data.ctl")

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "round-trip.ctl")
	if err := f.WriteToFile(outPath); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outPath, err)
	}

	roundTrip, err := lvrsrc.Parse(written, lvrsrc.OpenOptions{Strict: true})
	if err != nil {
		t.Fatalf("Parse(written) error = %v", err)
	}

	assertEquivalentFile(t, roundTrip, f)
}

func TestValidateReturnsNoIssuesForValidFixture(t *testing.T) {
	data := readFixture(t, "config-data.ctl")

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if issues := f.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %+v, want none", issues)
	}
}

func TestValidateReportsRecoverableHeaderMismatch(t *testing.T) {
	data := readFixture(t, "config-data.ctl")
	infoOffset := int(readU32BE(t, data, 16))
	data[infoOffset+8] ^= 0x01

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assertHasIssueCode(t, f.Validate(), "header.mismatch")
}

func assertHasIssueCode(t *testing.T, issues []lvrsrc.Issue, want string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("issue code %q not found in %+v", want, issues)
}

func assertEquivalentFile(t *testing.T, got, want *lvrsrc.File) {
	t.Helper()

	if got == nil || want == nil {
		t.Fatalf("got nil comparison: got=%v want=%v", got, want)
	}

	if got.Header != want.Header {
		t.Fatalf("Header = %#v, want %#v", got.Header, want.Header)
	}
	if got.SecondaryHeader != want.SecondaryHeader {
		t.Fatalf("SecondaryHeader = %#v, want %#v", got.SecondaryHeader, want.SecondaryHeader)
	}
	if got.BlockInfoList.DatasetInt1 != want.BlockInfoList.DatasetInt1 ||
		got.BlockInfoList.DatasetInt2 != want.BlockInfoList.DatasetInt2 ||
		got.BlockInfoList.DatasetInt3 != want.BlockInfoList.DatasetInt3 ||
		got.BlockInfoList.BlockInfoOffset != want.BlockInfoList.BlockInfoOffset {
		t.Fatalf("BlockInfoList = %#v, want compatible with %#v", got.BlockInfoList, want.BlockInfoList)
	}
	if got.Kind != want.Kind {
		t.Fatalf("Kind = %v, want %v", got.Kind, want.Kind)
	}
	if got.Compression != want.Compression {
		t.Fatalf("Compression = %v, want %v", got.Compression, want.Compression)
	}
	if len(got.Names) != len(want.Names) {
		t.Fatalf("len(Names) = %d, want %d", len(got.Names), len(want.Names))
	}
	for i := range want.Names {
		if got.Names[i] != want.Names[i] {
			t.Fatalf("Names[%d] = %#v, want %#v", i, got.Names[i], want.Names[i])
		}
	}
	if !bytes.Equal(got.RawTail, want.RawTail) {
		t.Fatalf("RawTail = %x, want %x", got.RawTail, want.RawTail)
	}
	if len(got.Blocks) != len(want.Blocks) {
		t.Fatalf("len(Blocks) = %d, want %d", len(got.Blocks), len(want.Blocks))
	}

	for bi := range want.Blocks {
		gotBlock := got.Blocks[bi]
		wantBlock := want.Blocks[bi]

		if gotBlock.Type != wantBlock.Type {
			t.Fatalf("block[%d].Type = %q, want %q", bi, gotBlock.Type, wantBlock.Type)
		}
		if gotBlock.SectionCountMinusOne != wantBlock.SectionCountMinusOne {
			t.Fatalf("block[%d].SectionCountMinusOne = %d, want %d", bi, gotBlock.SectionCountMinusOne, wantBlock.SectionCountMinusOne)
		}
		if gotBlock.Offset != wantBlock.Offset {
			t.Fatalf("block[%d].Offset = %d, want %d", bi, gotBlock.Offset, wantBlock.Offset)
		}
		if len(gotBlock.Sections) != len(wantBlock.Sections) {
			t.Fatalf("len(block[%d].Sections) = %d, want %d", bi, len(gotBlock.Sections), len(wantBlock.Sections))
		}

		for si := range wantBlock.Sections {
			gotSection := gotBlock.Sections[si]
			wantSection := wantBlock.Sections[si]

			if gotSection.Index != wantSection.Index {
				t.Fatalf("block[%d].section[%d].Index = %d, want %d", bi, si, gotSection.Index, wantSection.Index)
			}
			if gotSection.NameOffset != wantSection.NameOffset {
				t.Fatalf("block[%d].section[%d].NameOffset = %d, want %d", bi, si, gotSection.NameOffset, wantSection.NameOffset)
			}
			if gotSection.Unknown3 != wantSection.Unknown3 {
				t.Fatalf("block[%d].section[%d].Unknown3 = %d, want %d", bi, si, gotSection.Unknown3, wantSection.Unknown3)
			}
			if gotSection.DataOffset != wantSection.DataOffset {
				t.Fatalf("block[%d].section[%d].DataOffset = %d, want %d", bi, si, gotSection.DataOffset, wantSection.DataOffset)
			}
			if gotSection.Unknown5 != wantSection.Unknown5 {
				t.Fatalf("block[%d].section[%d].Unknown5 = %d, want %d", bi, si, gotSection.Unknown5, wantSection.Unknown5)
			}
			if gotSection.Name != wantSection.Name {
				t.Fatalf("block[%d].section[%d].Name = %q, want %q", bi, si, gotSection.Name, wantSection.Name)
			}
			if !bytes.Equal(gotSection.Payload, wantSection.Payload) {
				t.Fatalf("block[%d].section[%d].Payload = %x, want %x", bi, si, gotSection.Payload, wantSection.Payload)
			}
		}
	}
}
