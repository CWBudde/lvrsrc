package lvvi

import "testing"

func TestPartIDString(t *testing.T) {
	tests := []struct {
		p    PartID
		want string
	}{
		{PartIDNone, "NO_PARTID"},
		{PartIDNameLabel, "NAME_LABEL"},
		{PartIDRingText, "RING_TEXT"},
		{PartIDBooleanText, "BOOLEAN_TEXT"},
		{PartIDCaption, "CAPTION"},
		{8010, "TYPE_DEFS_CONTROL"},
		{8015, "GRAPH_SCALE_LEGEND"},
		{99999, "Part(99999)"},
	}
	for _, tc := range tests {
		if got := tc.p.String(); got != tc.want {
			t.Errorf("PartID(%d).String() = %q, want %q", int32(tc.p), got, tc.want)
		}
	}
}

func TestPartIDIsLabel(t *testing.T) {
	labels := []PartID{PartIDNameLabel, PartIDRingText, PartIDBooleanText, PartIDCaption, PartIDUnitLabel, PartIDNumLabel}
	for _, p := range labels {
		if !p.IsLabel() {
			t.Errorf("PartID %s should be a label role", p)
		}
	}
	nonLabels := []PartID{PartIDNone, PartIDCosmetic, PartIDFrame, PartIDHousing, PartIDBooleanButton}
	for _, p := range nonLabels {
		if p.IsLabel() {
			t.Errorf("PartID %s should not be a label role", p)
		}
	}
}

// TestPartIDNamesConstantsAgree guards that every named PartID constant has
// the matching entry in the authoritative name map.
func TestPartIDNamesConstantsAgree(t *testing.T) {
	consts := map[PartID]string{
		PartIDNone: "NO_PARTID", PartIDCosmetic: "COSMETIC", PartIDHousing: "HOUSING",
		PartIDFrame: "FRAME", PartIDNumericText: "NUMERIC_TEXT", PartIDText: "TEXT",
		PartIDRingText: "RING_TEXT", PartIDRadix: "RADIX", PartIDNameLabel: "NAME_LABEL",
		PartIDBooleanButton: "BOOLEAN_BUTTON", PartIDBooleanText: "BOOLEAN_TEXT",
		PartIDDecoration: "DECORATION", PartIDBooleanTrueLbl: "BOOLEAN_TRUE_LABEL",
		PartIDBooleanFalsLbl: "BOOLEAN_FALSE_LABEL", PartIDUnitLabel: "UNIT_LABEL",
		PartIDMenuTitleLabel: "MENU_TITLE_LABEL", PartIDCaption: "CAPTION",
		PartIDTabCaption: "TAB_CAPTION", PartIDScaleName: "SCALE_NAME",
		PartIDNumLabel: "NUM_LABEL", PartIDTernaryText: "TERNARY_TEXT",
	}
	for p, want := range consts {
		if got := p.String(); got != want {
			t.Errorf("constant for %q resolves to %q", want, got)
		}
	}
}
