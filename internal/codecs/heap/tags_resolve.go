package heap

import "strings"

// This file implements pylabview's context-dependent heap-tag naming.
//
// LabVIEW heaps reuse the same integer tag namespace across many
// unrelated enum families. The meaning of a positive tag depends on the
// class of the nearest enclosing object (the node's "parent top class"):
// the same tag id resolves to different fields inside an SL__Image, an
// SL__textHair, an SL__TreeNode, and so on. pylabview models this with
// tagIdToEnum(tagId, parentNode) -> parentTopClassEn -> per-class tag
// enum, falling back to the generic OBJ_FIELD_TAGS.
//
// classTagListResolvers mirrors pylabview's CLASS_EN_TO_TAG_LIST_MAPPING
// (LVheap.py). Each entry maps an object class to the per-class tag enum
// that governs its child field tags. The generated *Tag.String() methods
// supply the names (and a "Type(N)" fallback for unknown values, which we
// detect via the parenthesis the same way the rest of the package does).
//
// Keep this table in sync with CLASS_EN_TO_TAG_LIST_MAPPING when the
// generated enums in tags_gen.go are regenerated from a newer pylabview.
var classTagListResolvers = map[ClassTag]func(int32) string{
	ClassTagFontRun:            func(v int32) string { return FontRunTag(v).String() },
	ClassTagTextHair:           func(v int32) string { return TextHairTag(v).String() },
	ClassTagImage:              func(v int32) string { return ImageTag(v).String() },
	ClassTagSubCosm:            func(v int32) string { return SubcosmTag(v).String() },
	ClassTagEmbedObject:        func(v int32) string { return EmbedObjectTag(v).String() },
	ClassTagSceneView:          func(v int32) string { return SceneGraphTag(v).String() },
	ClassTagSceneColor:         func(v int32) string { return SceneColorTag(v).String() },
	ClassTagSceneEyePoint:      func(v int32) string { return SceneEyePointTag(v).String() },
	ClassTagComplexScalar:      func(v int32) string { return ComplexScalarTag(v).String() },
	ClassTagTableAttribute:     func(v int32) string { return AttributeListItemTag(v).String() },
	ClassTagTime128:            func(v int32) string { return Time128Tag(v).String() },
	ClassTagBrowseOptions:      func(v int32) string { return BrowseOptionsTag(v).String() },
	ClassTagStorageRowCol:      func(v int32) string { return RowColTag(v).String() },
	ClassTagColorPair:          func(v int32) string { return ColorPairTag(v).String() },
	ClassTagTreeNode:           func(v int32) string { return TreeNodeTag(v).String() },
	ClassTagRelativeRowCol:     func(v int32) string { return RowColTag(v).String() },
	ClassTagTabInfoItem:        func(v int32) string { return TabInfoItemTag(v).String() },
	ClassTagPageInfoItem:       func(v int32) string { return PageInfoItemTag(v).String() },
	ClassTagMappedPoint:        func(v int32) string { return MappedPointTag(v).String() },
	ClassTagPlotData:           func(v int32) string { return PlotDataTag(v).String() },
	ClassTagCursorData:         func(v int32) string { return CursorDataTag(v).String() },
	ClassTagPlotImages:         func(v int32) string { return PlotImagesTag(v).String() },
	ClassTagCursorButtonsRec:   func(v int32) string { return CursButtonsRecTag(v).String() },
	ClassTagPlotLegendData:     func(v int32) string { return PlotLegendDataTag(v).String() },
	ClassTagDigitlaBusOrgClust: func(v int32) string { return DigitalBusOrgClustTag(v).String() },
	ClassTagScaleLegendData:    func(v int32) string { return ScaleLegendDataTag(v).String() },
	ClassTagKeyMappingBinding:  func(v int32) string { return KeyMappingTag(v).String() },
	ClassTagScaleData:          func(v int32) string { return ScaleDataTag(v).String() },
	ClassTagConpaneConnection:  func(v int32) string { return ConnectionTag(v).String() },
	ClassTagGrowTermInfo:       func(v int32) string { return GrowTermInfoTag(v).String() },
	ClassTagEventSpec:          func(v int32) string { return EventSpecTag(v).String() },
	ClassTagSelectorRange:      func(v int32) string { return SelectorRangeTag(v).String() },
}

// ClassDefault is pylabview's parentTopClassEn fallback (SL__oHExt):
// the class assumed for a node when no enclosing object carries an
// SL__class attribute. SL__oHExt is not in classTagListResolvers, so it
// resolves field tags through the generic OBJ_FIELD_TAGS family.
const ClassDefault = ClassTagOHExt

// ResolveTagName resolves a heap node's tag to its symbolic name in the
// context of parentClass — the class of the nearest enclosing object
// (the node's parent top class). It mirrors pylabview's tagIdToEnum:
//
//  1. Negative tags are system tags (SL_SYSTEM_TAGS), context-free.
//  2. Positive tags resolve in parentClass's per-class tag list when one
//     exists (classTagListResolvers), e.g. tag 0 inside an SL__Image is
//     OF__ImageResID.
//  3. Otherwise they resolve in the generic OBJ_FIELD_TAGS family.
//
// The returned name keeps its OF__/SL__ prefix (matching the rest of the
// package and the generated enums). family is "system" or "field"; ok is
// false when no family recognises the tag, in which case name is empty
// and callers supply their own numeric fallback.
//
// Note: unlike the older context-free resolver, ResolveTagName never
// returns an SL_CLASS_TAGS name for a node's own tag. A node's class
// identity comes from its SL__class attribute, not its tag; the tag
// always names the node's role within its parent.
func ResolveTagName(tagID int32, parentClass ClassTag) (name, family string, ok bool) {
	if tagID < 0 {
		if n := SystemTag(tagID).String(); !strings.Contains(n, "(") {
			return n, "system", true
		}
		return "", "", false
	}
	if resolve, has := classTagListResolvers[parentClass]; has {
		if n := resolve(tagID); !strings.Contains(n, "(") {
			return n, "field", true
		}
	}
	if n := FieldTag(tagID).String(); !strings.Contains(n, "(") {
		return n, "field", true
	}
	return "", "", false
}
