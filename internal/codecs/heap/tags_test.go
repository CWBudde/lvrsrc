package heap

import (
	"strings"
	"testing"
)

// TestSystemTagSpotChecks verifies the core SL_SYSTEM_TAGS values match
// pylabview LVheap.py:58-63. These are the structural tags every heap
// stream uses; getting them right is non-negotiable.
func TestSystemTagSpotChecks(t *testing.T) {
	cases := []struct {
		got  SystemTag
		want int
		name string
	}{
		{SystemTagObject, -3, "SL__object"},
		{SystemTagArray, -4, "SL__array"},
		{SystemTagReference, -5, "SL__reference"},
		{SystemTagArrayElement, -6, "SL__arrayElement"},
		{SystemTagRootObject, -7, "SL__rootObject"},
	}
	for _, tc := range cases {
		if int(tc.got) != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, int(tc.got), tc.want)
		}
		if got := tc.got.String(); got != tc.name {
			t.Errorf("%s.String() = %q, want %q", tc.name, got, tc.name)
		}
	}
}

func TestSystemAttribTagSpotChecks(t *testing.T) {
	cases := []struct {
		got  SystemAttribTag
		want int
		name string
	}{
		{SystemAttribTagClass, -2, "SL__class"},
		{SystemAttribTagUid, -3, "SL__uid"},
		{SystemAttribTagStockObj, -4, "SL__stockObj"},
		{SystemAttribTagElements, -5, "SL__elements"},
		{SystemAttribTagIndex, -6, "SL__index"},
		{SystemAttribTagStockSource, -7, "SL__stockSource"},
	}
	for _, tc := range cases {
		if int(tc.got) != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, int(tc.got), tc.want)
		}
	}
}

func TestNodeScopeMatchesPylabview(t *testing.T) {
	if int(NodeScopeTagOpen) != 0 {
		t.Errorf("NodeScopeTagOpen = %d, want 0", int(NodeScopeTagOpen))
	}
	if int(NodeScopeTagLeaf) != 1 {
		t.Errorf("NodeScopeTagLeaf = %d, want 1", int(NodeScopeTagLeaf))
	}
	if int(NodeScopeTagClose) != 2 {
		t.Errorf("NodeScopeTagClose = %d, want 2", int(NodeScopeTagClose))
	}
}

func TestObjFieldTagSpotChecksLowFew(t *testing.T) {
	cases := []struct {
		got  FieldTag
		want int
		name string
	}{
		{FieldTagActiveDiag, 1, "OF__activeDiag"},
		{FieldTagActiveMarker, 2, "OF__activeMarker"},
		{FieldTagBgColor, 9, "OF__bgColor"},
	}
	for _, tc := range cases {
		if int(tc.got) != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, int(tc.got), tc.want)
		}
		if got := tc.got.String(); got != tc.name {
			t.Errorf("%s.String() = %q, want %q", tc.name, got, tc.name)
		}
	}
}

func TestStringFallbackForUnknownValue(t *testing.T) {
	// A wildly out-of-range FieldTag must not panic and must produce a
	// `FieldTag(N)` style fallback so log lines remain readable.
	got := FieldTag(99999).String()
	if !strings.HasPrefix(got, "FieldTag(") {
		t.Errorf("unknown FieldTag.String() = %q, want FieldTag(99999)-like fallback", got)
	}
}

// TestPylavbiewAliasIsPreservedInConsts verifies the two members that
// share a numeric value in pylabview (OF__tagDLLPath = OF__recursiveFunc
// = 430) both compile as constants. Only one of the two names will
// reverse-lookup via String() — that's the canonical name pick.
func TestPylavbiewAliasIsPreservedInConsts(t *testing.T) {
	if int(FieldTagTagDLLPath) != 430 {
		t.Errorf("FieldTagTagDLLPath = %d, want 430", int(FieldTagTagDLLPath))
	}
	if int(FieldTagRecursiveFunc) != 430 {
		t.Errorf("FieldTagRecursiveFunc = %d, want 430", int(FieldTagRecursiveFunc))
	}
	got := FieldTag(430).String()
	// Whichever was declared first in pylabview wins. We don't assert
	// which one — only that it round-trips to one of the two names.
	if got != "OF__tagDLLPath" && got != "OF__recursiveFunc" {
		t.Errorf("FieldTag(430).String() = %q, want one of OF__tagDLLPath / OF__recursiveFunc", got)
	}
}

// TestCaseSensitiveDuplicatesEachExist confirms that pylabview's two
// case-distinct members (OF__commentSelLabData = 522 and
// OF__CommentSelLabData = 586) survive the port as separate constants
// with their original numeric values.
func TestCaseSensitiveDuplicatesEachExist(t *testing.T) {
	if int(FieldTagCommentSelLabData) != 522 {
		t.Errorf("FieldTagCommentSelLabData = %d, want 522", int(FieldTagCommentSelLabData))
	}
	if int(FieldTagCommentSelLabData2) != 586 {
		t.Errorf("FieldTagCommentSelLabData2 = %d, want 586", int(FieldTagCommentSelLabData2))
	}
}

func TestHeapFormatValues(t *testing.T) {
	cases := []struct {
		got  HeapFormat
		want int
		name string
	}{
		{HeapFormatUnknown, 0, "Unknown"},
		{HeapFormatVersionT, 1, "VersionT"},
		{HeapFormatXMLVer, 2, "XMLVer"},
		{HeapFormatBinVerA, 3, "BinVerA"},
		{HeapFormatBinVerB, 4, "BinVerB"},
		{HeapFormatBinVerC, 5, "BinVerC"},
	}
	for _, tc := range cases {
		if int(tc.got) != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, int(tc.got), tc.want)
		}
	}
}
