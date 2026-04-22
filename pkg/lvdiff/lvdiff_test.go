package lvdiff

import (
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func baseFile() *lvrsrc.File {
	return &lvrsrc.File{
		Header: lvrsrc.Header{
			Magic:         "RSRC",
			FormatVersion: 3,
			Type:          "LVIN",
			Creator:       "LBVW",
			InfoOffset:    0x20,
			InfoSize:      0x100,
			DataOffset:    0x200,
			DataSize:      0x400,
		},
		SecondaryHeader: lvrsrc.Header{
			Magic:         "RSRC",
			FormatVersion: 3,
			Type:          "LVIN",
			Creator:       "LBVW",
			InfoOffset:    0x20,
			InfoSize:      0x100,
			DataOffset:    0x200,
			DataSize:      0x400,
		},
		BlockInfoList: lvrsrc.BlockInfoList{
			BlockInfoOffset: 0x30,
			BlockInfoSize:   0x40,
		},
		Blocks: []lvrsrc.Block{
			{
				Type: "LVSR",
				Sections: []lvrsrc.Section{
					{Index: 0, Name: "", Payload: []byte{0x01, 0x02, 0x03, 0x04}},
				},
			},
			{
				Type: "icl8",
				Sections: []lvrsrc.Section{
					{Index: 0, Name: "icon", Payload: []byte{0xAA, 0xBB}},
				},
			},
		},
		Kind: lvrsrc.FileKindVI,
	}
}

func TestFilesIdenticalHasEmptyDiff(t *testing.T) {
	a := baseFile()
	b := baseFile()

	d := Files(a, b)
	if d == nil {
		t.Fatal("Files returned nil")
	}
	if !d.IsEmpty() {
		t.Fatalf("expected empty diff, got %d items: %+v", len(d.Items), d.Items)
	}
}

func TestFilesNilInputs(t *testing.T) {
	if d := Files(nil, nil); d == nil || !d.IsEmpty() {
		t.Fatalf("Files(nil, nil) = %+v, want empty diff", d)
	}

	d := Files(nil, baseFile())
	if d == nil || d.IsEmpty() {
		t.Fatalf("Files(nil, b) = %+v, want non-empty diff", d)
	}
	if got := d.Items[0].Category; got != CategoryAdded {
		t.Fatalf("Files(nil, b)[0].Category = %q, want %q", got, CategoryAdded)
	}

	d = Files(baseFile(), nil)
	if d == nil || d.IsEmpty() {
		t.Fatalf("Files(a, nil) = %+v, want non-empty diff", d)
	}
	if got := d.Items[0].Category; got != CategoryRemoved {
		t.Fatalf("Files(a, nil)[0].Category = %q, want %q", got, CategoryRemoved)
	}
}

func TestFilesHeaderFieldDiff(t *testing.T) {
	a := baseFile()
	b := baseFile()
	b.Header.FormatVersion = 4
	b.Header.DataSize = 0x500

	d := Files(a, b)

	headerDiffs := d.Filter(KindHeader)
	if len(headerDiffs) != 2 {
		t.Fatalf("expected 2 header diffs, got %d: %+v", len(headerDiffs), headerDiffs)
	}

	paths := map[string]DiffItem{}
	for _, it := range headerDiffs {
		paths[it.Path] = it
	}

	fv, ok := paths["header.FormatVersion"]
	if !ok {
		t.Fatalf("expected path header.FormatVersion in diffs: %+v", paths)
	}
	if fv.Category != CategoryModified {
		t.Fatalf("FormatVersion diff category = %q, want modified", fv.Category)
	}
	if fv.Old != uint16(3) || fv.New != uint16(4) {
		t.Fatalf("FormatVersion diff Old=%v New=%v, want 3 / 4", fv.Old, fv.New)
	}

	if _, ok := paths["header.DataSize"]; !ok {
		t.Fatalf("expected path header.DataSize in diffs: %+v", paths)
	}
}

func TestFilesSecondaryHeaderAndBlockInfoListDiff(t *testing.T) {
	a := baseFile()
	b := baseFile()
	b.SecondaryHeader.InfoOffset = 0x99
	b.BlockInfoList.BlockInfoSize = 0x41

	d := Files(a, b)

	want := map[string]bool{
		"secondaryHeader.InfoOffset":  false,
		"blockInfoList.BlockInfoSize": false,
	}
	for _, it := range d.Filter(KindHeader) {
		if _, ok := want[it.Path]; ok {
			want[it.Path] = true
		}
	}
	for path, seen := range want {
		if !seen {
			t.Fatalf("expected diff for path %q, got items=%+v", path, d.Items)
		}
	}
}

func TestFilesResourceTypeAddedAndRemoved(t *testing.T) {
	a := baseFile()
	b := baseFile()
	// Remove icl8, add BDPW.
	b.Blocks = []lvrsrc.Block{
		b.Blocks[0],
		{
			Type: "BDPW",
			Sections: []lvrsrc.Section{
				{Index: 0, Payload: []byte{0xDE, 0xAD}},
			},
		},
	}

	d := Files(a, b)

	blockDiffs := d.Filter(KindBlock)
	var added, removed []DiffItem
	for _, it := range blockDiffs {
		switch it.Category {
		case CategoryAdded:
			added = append(added, it)
		case CategoryRemoved:
			removed = append(removed, it)
		}
	}
	if len(added) != 1 || added[0].Path != "blocks.BDPW" {
		t.Fatalf("expected 1 added block BDPW, got %+v", added)
	}
	if len(removed) != 1 || removed[0].Path != "blocks.icl8" {
		t.Fatalf("expected 1 removed block icl8, got %+v", removed)
	}
}

func TestFilesSectionSizeDiff(t *testing.T) {
	a := baseFile()
	b := baseFile()
	b.Blocks[0].Sections[0].Payload = []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	d := Files(a, b)

	sectionDiffs := d.Filter(KindSection)
	if len(sectionDiffs) == 0 {
		t.Fatalf("expected at least one section diff, got none")
	}

	var sizeItem *DiffItem
	for i := range sectionDiffs {
		if sectionDiffs[i].Path == "blocks.LVSR/0.size" {
			sizeItem = &sectionDiffs[i]
			break
		}
	}
	if sizeItem == nil {
		t.Fatalf("expected blocks.LVSR/0.size diff item, got %+v", sectionDiffs)
	}
	if sizeItem.Old != 4 || sizeItem.New != 5 {
		t.Fatalf("size diff Old=%v New=%v, want 4 / 5", sizeItem.Old, sizeItem.New)
	}
}

func TestFilesSectionContentHashDiff(t *testing.T) {
	a := baseFile()
	b := baseFile()
	// Same size, different content.
	b.Blocks[0].Sections[0].Payload = []byte{0x09, 0x09, 0x09, 0x09}

	d := Files(a, b)

	var contentItem *DiffItem
	for i := range d.Items {
		if d.Items[i].Path == "blocks.LVSR/0.content" {
			contentItem = &d.Items[i]
			break
		}
	}
	if contentItem == nil {
		t.Fatalf("expected blocks.LVSR/0.content diff item, got %+v", d.Items)
	}

	oldHash, ok := contentItem.Old.(string)
	if !ok || len(oldHash) == 0 {
		t.Fatalf("content diff Old = %v, want non-empty hex string", contentItem.Old)
	}
	newHash, ok := contentItem.New.(string)
	if !ok || len(newHash) == 0 {
		t.Fatalf("content diff New = %v, want non-empty hex string", contentItem.New)
	}
	if oldHash == newHash {
		t.Fatalf("content diff hashes equal: %q", oldHash)
	}
}

func TestFilesSectionIdenticalPayloadNoContentDiff(t *testing.T) {
	a := baseFile()
	b := baseFile()
	// Touch Name but payload equal; we should not emit a content diff.
	b.Blocks[0].Sections[0].Name = "Renamed"

	d := Files(a, b)
	for _, it := range d.Items {
		if it.Path == "blocks.LVSR/0.content" {
			t.Fatalf("unexpected content diff for unchanged payload: %+v", it)
		}
		if it.Path == "blocks.LVSR/0.size" {
			t.Fatalf("unexpected size diff for unchanged payload: %+v", it)
		}
	}
}

func TestFilesSectionAddedAndRemoved(t *testing.T) {
	a := baseFile()
	b := baseFile()
	// Append a new section to LVSR.
	b.Blocks[0].Sections = append(b.Blocks[0].Sections, lvrsrc.Section{
		Index: 1, Payload: []byte{0xFF},
	})
	// Drop the icl8 section (but keep the block present with zero sections).
	b.Blocks[1].Sections = nil

	d := Files(a, b)

	var addedPath, removedPath bool
	for _, it := range d.Filter(KindSection) {
		if it.Path == "blocks.LVSR/1" && it.Category == CategoryAdded {
			addedPath = true
		}
		if it.Path == "blocks.icl8/0" && it.Category == CategoryRemoved {
			removedPath = true
		}
	}
	if !addedPath {
		t.Fatalf("expected added section blocks.LVSR/1, got items=%+v", d.Items)
	}
	if !removedPath {
		t.Fatalf("expected removed section blocks.icl8/0, got items=%+v", d.Items)
	}
}

func TestDiffByCategoryAndFilter(t *testing.T) {
	a := baseFile()
	b := baseFile()
	b.Header.FormatVersion = 4
	b.Blocks[1].Sections = nil

	d := Files(a, b)

	if d.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	if len(d.ByCategory(CategoryModified)) == 0 {
		t.Fatal("expected at least one modified item")
	}
	if len(d.ByCategory(CategoryRemoved)) == 0 {
		t.Fatal("expected at least one removed item")
	}
	if len(d.Filter(KindHeader)) == 0 {
		t.Fatal("expected at least one header diff")
	}
	if len(d.Filter(KindSection)) == 0 {
		t.Fatal("expected at least one section diff")
	}
}

func TestFilesDecodedStubExtensionPoint(t *testing.T) {
	// The decoded-resource diff is a stub until Phase 4+. Calling Files with
	// a DecodedDiffer option should invoke the differ for each section pair
	// whose block type is registered, and include its items in the result.
	a := baseFile()
	b := baseFile()
	b.Blocks[0].Sections[0].Payload = []byte{0x01, 0x02, 0x03, 0x05}

	calls := 0
	opts := Options{
		DecodedDiffers: map[string]DecodedDiffer{
			"LVSR": func(blockType string, sectionIndex int32, oldPayload, newPayload []byte) []DiffItem {
				calls++
				return []DiffItem{{
					Kind:     KindDecoded,
					Category: CategoryModified,
					Path:     "blocks.LVSR/0.decoded.someField",
					Old:      "a",
					New:      "b",
					Message:  "test differ",
				}}
			},
		},
	}

	d := FilesWithOptions(a, b, opts)
	if calls != 1 {
		t.Fatalf("decoded differ called %d times, want 1", calls)
	}
	decoded := d.Filter(KindDecoded)
	if len(decoded) != 1 {
		t.Fatalf("expected 1 decoded diff item, got %d: %+v", len(decoded), decoded)
	}
	if decoded[0].Path != "blocks.LVSR/0.decoded.someField" {
		t.Fatalf("decoded diff path = %q, want blocks.LVSR/0.decoded.someField", decoded[0].Path)
	}
}
