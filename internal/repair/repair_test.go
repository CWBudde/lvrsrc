package repair

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/rsrcwire"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestFileRepairsHeaderMismatch(t *testing.T) {
	f := parseLenientFixture(t, corruptSecondaryHeaderMismatch(t))

	repaired, actions, err := File(f)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if got, want := len(actions), 1; got != want {
		t.Fatalf("len(actions) = %d, want %d", got, want)
	}
	if actions[0] != "rewrite headers from parsed structure" {
		t.Fatalf("actions[0] = %q", actions[0])
	}

	if repaired == f {
		t.Fatal("File() returned original pointer")
	}
}

func TestFileRepairsOffsetOverlapByRewrite(t *testing.T) {
	f := parseLenientFixture(t, corruptPayloadOverlap(t))

	repaired, actions, err := File(f)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("actions = empty, want non-empty")
	}

	var sawOverlap bool
	for _, action := range actions {
		if action == "recompute section/header offsets from parsed payload tree" {
			sawOverlap = true
		}
	}
	if !sawOverlap {
		t.Fatalf("actions = %v, want overlap repair action", actions)
	}

	if issues := repaired.Validate(); !hasIssueCode(issues, "section.payload.overlap") {
		t.Fatalf("expected input issue to still be present before serialization, issues=%+v", issues)
	}
}

func TestFileRefusesUnresolvedTruncatedNameTable(t *testing.T) {
	f := parseLenientFixture(t, corruptMissingSectionName(t))

	_, _, err := File(f)
	if err == nil {
		t.Fatal("File() error = nil, want non-nil")
	}
	if got := err.Error(); got == "" || !containsAll(got, "missing section name", "section.name_offset.invalid") {
		t.Fatalf("File() error = %q, want missing-name refusal", got)
	}
}

func parseLenientFixture(t *testing.T, path string) *lvrsrc.File {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{Strict: false})
	if err != nil {
		t.Fatalf("Parse(lenient) error = %v", err)
	}
	return f
}

func corruptSecondaryHeaderMismatch(t *testing.T) string {
	t.Helper()
	data := readCorpusFixture(t, "config-data.ctl")
	infoOffset := binary.BigEndian.Uint32(data[16:20])
	data[int(infoOffset)+8] ^= 0x01
	return writeTempFixture(t, "header-mismatch.ctl", data)
}

func corruptPayloadOverlap(t *testing.T) string {
	t.Helper()
	data := readCorpusFixture(t, "config-data.ctl")
	f, err := rsrcwire.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(f.Blocks) < 2 || len(f.Blocks[0].Sections) == 0 || len(f.Blocks[1].Sections) == 0 {
		t.Fatal("fixture shape changed; need at least two blocks with one section each")
	}

	blockInfoPos := int(f.Header.InfoOffset + f.BlockInfoList.BlockInfoOffset)
	secondSectionPos := blockInfoPos + int(f.Blocks[1].Offset)
	binary.BigEndian.PutUint32(data[secondSectionPos+12:], f.Blocks[0].Sections[0].DataOffset)
	return writeTempFixture(t, "payload-overlap.ctl", data)
}

func corruptMissingSectionName(t *testing.T) string {
	t.Helper()
	data := readCorpusFixture(t, "config-data.ctl")
	f, err := rsrcwire.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	blockIndex, sectionIndex, ok := firstNamedSection(f)
	if !ok {
		t.Fatal("fixture has no named sections")
	}

	blockInfoPos := int(f.Header.InfoOffset + f.BlockInfoList.BlockInfoOffset)
	sectionPos := blockInfoPos + int(f.Blocks[blockIndex].Offset) + sectionIndex*20
	binary.BigEndian.PutUint32(data[sectionPos+4:], f.Header.InfoSize)
	return writeTempFixture(t, "missing-name.ctl", data)
}

func firstNamedSection(f *rsrcwire.File) (int, int, bool) {
	for bi, block := range f.Blocks {
		for si, section := range block.Sections {
			if section.NameOffset != ^uint32(0) {
				return bi, si, true
			}
		}
	}
	return 0, 0, false
}

func readCorpusFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(corpus.Path(name))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", name, err)
	}
	return data
}

func writeTempFixture(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := fmt.Sprintf("%s/%s", t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}

func hasIssueCode(issues []lvrsrc.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
