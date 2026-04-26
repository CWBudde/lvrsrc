package heap

import (
	"fmt"
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

// TestGeneratedStringers exercises every enum String() in tags_gen.go on
// both branches: a known value (one map entry per type) plus a sentinel that
// is guaranteed not to appear so the `Type(N)` fallback runs. This keeps the
// auto-generated stringers from dominating the heap package's coverage gap.
func TestGeneratedStringers(t *testing.T) {
	const sentinel = 999_999
	cases := []struct {
		typeName string
		named    fmt.Stringer
		fallback fmt.Stringer
	}{
		{"HeapFormat", HeapFormatVersionT, HeapFormat(sentinel)},
		{"NodeScope", NodeScopeTagOpen, NodeScope(sentinel)},
		{"SystemTag", SystemTagObject, SystemTag(sentinel)},
		{"SystemAttribTag", SystemAttribTagClass, SystemAttribTag(sentinel)},
		{"FieldTag", FieldTagActiveDiag, FieldTag(sentinel)},
		{"ClassTag", anyKey(classTagNames), ClassTag(sentinel)},
		{"MultiDimClassTag", MultiDimClassTagMultiDimArray, MultiDimClassTag(sentinel)},
		{"FontRunTag", anyKey(fontRunTagNames), FontRunTag(sentinel)},
		{"TextHairTag", anyKey(textHairTagNames), TextHairTag(sentinel)},
		{"ComplexScalarTag", anyKey(complexScalarTagNames), ComplexScalarTag(sentinel)},
		{"Time128Tag", anyKey(time128TagNames), Time128Tag(sentinel)},
		{"ImageTag", anyKey(imageTagNames), ImageTag(sentinel)},
		{"SubcosmTag", anyKey(subcosmTagNames), SubcosmTag(sentinel)},
		{"EmbedObjectTag", anyKey(embedObjectTagNames), EmbedObjectTag(sentinel)},
		{"SceneGraphTag", anyKey(sceneGraphTagNames), SceneGraphTag(sentinel)},
		{"SceneColorTag", anyKey(sceneColorTagNames), SceneColorTag(sentinel)},
		{"SceneEyePointTag", anyKey(sceneEyePointTagNames), SceneEyePointTag(sentinel)},
		{"AttributeListItemTag", anyKey(attributeListItemTagNames), AttributeListItemTag(sentinel)},
		{"BrowseOptionsTag", anyKey(browseOptionsTagNames), BrowseOptionsTag(sentinel)},
		{"RowColTag", anyKey(rowColTagNames), RowColTag(sentinel)},
		{"ColorPairTag", anyKey(colorPairTagNames), ColorPairTag(sentinel)},
		{"TreeNodeTag", anyKey(treeNodeTagNames), TreeNodeTag(sentinel)},
		{"TabInfoItemTag", anyKey(tabInfoItemTagNames), TabInfoItemTag(sentinel)},
		{"PageInfoItemTag", anyKey(pageInfoItemTagNames), PageInfoItemTag(sentinel)},
		{"MappedPointTag", anyKey(mappedPointTagNames), MappedPointTag(sentinel)},
		{"PlotDataTag", anyKey(plotDataTagNames), PlotDataTag(sentinel)},
		{"CursorDataTag", anyKey(cursorDataTagNames), CursorDataTag(sentinel)},
		{"PlotImagesTag", anyKey(plotImagesTagNames), PlotImagesTag(sentinel)},
		{"CursButtonsRecTag", anyKey(cursButtonsRecTagNames), CursButtonsRecTag(sentinel)},
		{"PlotLegendDataTag", anyKey(plotLegendDataTagNames), PlotLegendDataTag(sentinel)},
		{"DigitalBusOrgClustTag", anyKey(digitalBusOrgClustTagNames), DigitalBusOrgClustTag(sentinel)},
		{"ScaleLegendDataTag", anyKey(scaleLegendDataTagNames), ScaleLegendDataTag(sentinel)},
		{"ScaleDataTag", anyKey(scaleDataTagNames), ScaleDataTag(sentinel)},
		{"KeyMappingTag", anyKey(keyMappingTagNames), KeyMappingTag(sentinel)},
		{"MultiDimTag", anyKey(multiDimTagNames), MultiDimTag(sentinel)},
		{"GrowTermInfoTag", anyKey(growTermInfoTagNames), GrowTermInfoTag(sentinel)},
		{"ConnectionTag", anyKey(connectionTagNames), ConnectionTag(sentinel)},
		{"SelectorRangeTag", anyKey(selectorRangeTagNames), SelectorRangeTag(sentinel)},
		{"EventSpecTag", anyKey(eventSpecTagNames), EventSpecTag(sentinel)},
		{"BaseTableControlFlagsTag", anyKey(baseTableControlFlagsTagNames), BaseTableControlFlagsTag(sentinel)},
		{"BaseListboxFlagsTag", anyKey(baseListboxFlagsTagNames), BaseListboxFlagsTag(sentinel)},
	}

	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			named := tc.named.String()
			if named == "" {
				t.Errorf("named %s.String() returned empty", tc.typeName)
			}
			if strings.HasPrefix(named, tc.typeName+"(") {
				t.Errorf("named %s.String() unexpectedly fell back: %q", tc.typeName, named)
			}
			fb := tc.fallback.String()
			wantPrefix := tc.typeName + "("
			if !strings.HasPrefix(fb, wantPrefix) {
				t.Errorf("fallback %s.String() = %q, want %s… prefix", tc.typeName, fb, wantPrefix)
			}
		})
	}
}

// anyKey returns an arbitrary key from a generated stringer map, panicking
// if the map is empty (which would indicate a regression in the generator).
// Only used to dodge hardcoding a per-type member name in the test table.
func anyKey[K comparable, V any](m map[K]V) K {
	for k := range m {
		return k
	}
	panic("empty stringer map")
}
