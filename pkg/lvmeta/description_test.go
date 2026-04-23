package lvmeta

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestSetDescriptionUpdatesExistingSTRGInPlace(t *testing.T) {
	f := fileWithSingleSTRG(t, "old")

	if err := SetDescription(f, "hello world"); err != nil {
		t.Fatalf("SetDescription err = %v, want nil", err)
	}

	section := f.Blocks[1].Sections[0]
	if got := string(section.Payload[4:]); got != "hello world" {
		t.Fatalf("STRG text = %q, want %q", got, "hello world")
	}
	size := binary.BigEndian.Uint32(section.Payload[:4])
	if size != uint32(len("hello world")) {
		t.Fatalf("STRG size prefix = %d, want %d", size, len("hello world"))
	}

	if len(f.Blocks) != 2 {
		t.Fatalf("len(Blocks) = %d, want 2 (no new block created)", len(f.Blocks))
	}
}

func TestSetDescriptionCreatesNewSTRGWhenMissing(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Kind:   lvrsrc.FileKindVI,
		Blocks: []lvrsrc.Block{
			{
				Type: "OPQ ",
				Sections: []lvrsrc.Section{
					{Index: 0, Payload: []byte{0xAA, 0xBB}},
				},
			},
		},
	}

	if err := SetDescription(f, "brand new"); err != nil {
		t.Fatalf("SetDescription err = %v, want nil", err)
	}

	if len(f.Blocks) != 2 {
		t.Fatalf("len(Blocks) = %d, want 2 (existing + newly created STRG)", len(f.Blocks))
	}
	newBlock := f.Blocks[1]
	if newBlock.Type != strg.FourCC {
		t.Fatalf("new block Type = %q, want %q", newBlock.Type, strg.FourCC)
	}
	if len(newBlock.Sections) != 1 {
		t.Fatalf("new block has %d sections, want 1", len(newBlock.Sections))
	}
	section := newBlock.Sections[0]
	if section.Index != 0 {
		t.Fatalf("new section Index = %d, want 0 (deterministic)", section.Index)
	}
	if section.Name != "" {
		t.Fatalf("new section Name = %q, want empty", section.Name)
	}
	if section.NameOffset != ^uint32(0) {
		t.Fatalf("new section NameOffset = %#x, want 0xFFFFFFFF (no-name sentinel)", section.NameOffset)
	}
	if got := string(section.Payload[4:]); got != "brand new" {
		t.Fatalf("new STRG text = %q, want %q", got, "brand new")
	}

	// Existing opaque block must not have moved or changed.
	if f.Blocks[0].Type != "OPQ " {
		t.Fatalf("existing block[0].Type = %q, want %q", f.Blocks[0].Type, "OPQ ")
	}
	if !bytesEqual(f.Blocks[0].Sections[0].Payload, []byte{0xAA, 0xBB}) {
		t.Fatalf("existing opaque payload changed")
	}
}

func TestSetDescriptionRejectsAmbiguousMultipleSTRG(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{
				Type: "STRG",
				Sections: []lvrsrc.Section{
					{Index: 1, Payload: strgPayload(t, "a")},
					{Index: 2, Payload: strgPayload(t, "b")},
				},
			},
		},
	}

	err := SetDescription(f, "ambig")
	if !errors.Is(err, ErrTargetAmbiguous) {
		t.Fatalf("err = %v, want ErrTargetAmbiguous", err)
	}
	// Original sections must not have been mutated.
	if got := string(f.Blocks[0].Sections[0].Payload[4:]); got != "a" {
		t.Fatalf("section[0] mutated: %q", got)
	}
	if got := string(f.Blocks[0].Sections[1].Payload[4:]); got != "b" {
		t.Fatalf("section[1] mutated: %q", got)
	}
}

func TestSetDescriptionEmptyProducesZeroLengthPayload(t *testing.T) {
	f := fileWithSingleSTRG(t, "old")

	if err := SetDescription(f, ""); err != nil {
		t.Fatalf("SetDescription empty err = %v, want nil", err)
	}

	payload := f.Blocks[1].Sections[0].Payload
	if len(payload) != 4 {
		t.Fatalf("empty payload len = %d, want 4 (size header only)", len(payload))
	}
	if size := binary.BigEndian.Uint32(payload[:4]); size != 0 {
		t.Fatalf("empty payload size prefix = %d, want 0", size)
	}
}

func TestSetDescriptionPreservesBytesVerbatim(t *testing.T) {
	// Includes CR, LF, CRLF, and a high-bit byte. None should be normalized.
	raw := "line1\nline2\r\nline3\r\xC3\xA9end"
	f := fileWithSingleSTRG(t, "something else")

	if err := SetDescription(f, raw); err != nil {
		t.Fatalf("SetDescription err = %v, want nil", err)
	}
	if got := string(f.Blocks[1].Sections[0].Payload[4:]); got != raw {
		t.Fatalf("payload text = %q, want %q", got, raw)
	}
}

func TestSetDescriptionNilFile(t *testing.T) {
	err := SetDescription(nil, "x")
	if !errors.Is(err, ErrNilFile) {
		t.Fatalf("err = %v, want ErrNilFile", err)
	}
}

func TestMutatorSetDescriptionStrictPropagates(t *testing.T) {
	// Strict Mutator should route through applyTypedEdit and inherit its
	// policy. For STRG+clean input, there are no warnings, so strict must
	// succeed.
	f := fileWithSingleSTRG(t, "old")
	if err := (Mutator{Strict: true}).SetDescription(f, "strict"); err != nil {
		t.Fatalf("strict SetDescription err = %v, want nil", err)
	}
	if got := string(f.Blocks[1].Sections[0].Payload[4:]); got != "strict" {
		t.Fatalf("payload = %q, want strict", got)
	}
}
