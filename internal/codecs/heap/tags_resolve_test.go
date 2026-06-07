package heap

import "testing"

func TestResolveTagNameContextDependent(t *testing.T) {
	tests := []struct {
		name       string
		tag        int32
		class      ClassTag
		wantName   string
		wantFamily string
		wantOK     bool
	}{
		// Same tag id, different parent class -> different field name.
		{"tag0 in Image", 0, ClassTagImage, "OF__ImageResID", "field", true},
		{"tag0 in SubCosm", 0, ClassTagSubCosm, "OF__Bounds", "field", true},
		{"tag0 in fontRun", 0, ClassTagFontRun, "OF__textRecObject", "field", true},
		// Generic field tags resolve regardless of class (not in any class list).
		{"objFlags generic", 172, ClassTagImage, "OF__objFlags", "field", true},
		{"howGrow generic", 106, ClassDefault, "OF__howGrow", "field", true},
		{"partID generic", 192, ClassTagSupC, "OF__partID", "field", true},
		// Class-specific tag falls back to generic OBJ_FIELD_TAGS when the
		// parent class has no per-class list (oHExt default).
		{"image-tag under default", 110, ClassDefault, "OF__image", "field", true},
		// Negative tags are context-free system tags.
		{"system arrayElement", -6, ClassTagImage, "SL__arrayElement", "system", true},
		{"system rootObject", -7, ClassDefault, "SL__rootObject", "system", true},
		// Unknown.
		{"unknown high tag", 99999, ClassDefault, "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, family, ok := ResolveTagName(tc.tag, tc.class)
			if ok != tc.wantOK || name != tc.wantName || family != tc.wantFamily {
				t.Errorf("ResolveTagName(%d, %v) = (%q, %q, %v), want (%q, %q, %v)",
					tc.tag, tc.class, name, family, ok, tc.wantName, tc.wantFamily, tc.wantOK)
			}
		})
	}
}

// TestResolveTagNameNeverReturnsClassName guards the key behavioural
// change: a node's own tag must never resolve to an SL_CLASS_TAGS name
// (those collide with OBJ_FIELD_TAGS values, e.g. 172 is both
// OF__objFlags and SL__grouper). The class identity comes from the
// SL__class attribute, not the tag.
func TestResolveTagNameNeverReturnsClassName(t *testing.T) {
	collisions := []struct {
		tag       int32
		className string // the SL__ name the old resolver wrongly returned
		wantField string
	}{
		{172, "SL__grouper", "OF__objFlags"},
		{106, "SL__extFunc", "OF__howGrow"},
		{192, "SL__exprNode", "OF__partID"},
		{144, "SL__scanfArg", "OF__masterPart"},
		{9, "SL__cosm", "OF__bgColor"},
		{80, "SL__stdNum", "OF__fgColor"},
	}
	for _, c := range collisions {
		name, family, ok := ResolveTagName(c.tag, ClassDefault)
		if !ok || family != "field" || name != c.wantField {
			t.Errorf("ResolveTagName(%d) = (%q, %q, %v), want field %q (not class %q)",
				c.tag, name, family, ok, c.wantField, c.className)
		}
	}
}
