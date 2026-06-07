package lvvi

import "fmt"

// PartID identifies the role of a part within a LabVIEW control or
// indicator — the integer stored in the OF__partID heap field. A control
// is assembled from many parts (a cosmetic frame, an increment button, a
// name label, the ring item text, …) and each carries a PartID so the
// editor can address it. Naming the value lets callers attribute heap text
// and cosmetics to a role (NAME_LABEL, CAPTION, RING_TEXT, …) instead of a
// bare integer.
//
// The vocabulary mirrors pylabview's PARTID enum
// (references/pylabview/pylabview/LVparts.py). partIDNames is the complete
// authoritative table; the named constants below cover the roles this
// package branches on plus the common label/text parts.
type PartID int32

// Named PartID constants. This is a curated subset of partIDNames chosen
// for the roles callers commonly reference (labels, captions, item text);
// PartID.String resolves the full set regardless of whether a constant
// exists.
const (
	PartIDNone           PartID = 0   // NO_PARTID
	PartIDCosmetic       PartID = 1   // COSMETIC
	PartIDHousing        PartID = 8   // HOUSING
	PartIDFrame          PartID = 9   // FRAME
	PartIDNumericText    PartID = 10  // NUMERIC_TEXT
	PartIDText           PartID = 11  // TEXT
	PartIDRingText       PartID = 12  // RING_TEXT — ring/enum item text list
	PartIDRadix          PartID = 15  // RADIX
	PartIDNameLabel      PartID = 16  // NAME_LABEL — the control's own caption
	PartIDBooleanButton  PartID = 21  // BOOLEAN_BUTTON
	PartIDBooleanText    PartID = 22  // BOOLEAN_TEXT — boolean state text list
	PartIDDecoration     PartID = 25  // DECORATION
	PartIDBooleanTrueLbl PartID = 63  // BOOLEAN_TRUE_LABEL
	PartIDBooleanFalsLbl PartID = 64  // BOOLEAN_FALSE_LABEL
	PartIDUnitLabel      PartID = 65  // UNIT_LABEL
	PartIDMenuTitleLabel PartID = 81  // MENU_TITLE_LABEL
	PartIDCaption        PartID = 82  // CAPTION
	PartIDTabCaption     PartID = 92  // TAB_CAPTION
	PartIDScaleName      PartID = 94  // SCALE_NAME
	PartIDNumLabel       PartID = 101 // NUM_LABEL
	PartIDTernaryText    PartID = 110 // TERNARY_TEXT
)

// IsLabel reports whether the part is a human-readable text label, caption,
// or item-text role — the parts whose heap content is displayed text.
func (p PartID) IsLabel() bool {
	switch p {
	case PartIDRingText, PartIDNameLabel, PartIDBooleanText,
		PartIDBooleanTrueLbl, PartIDBooleanFalsLbl, PartIDUnitLabel,
		PartIDMenuTitleLabel, PartIDCaption, PartIDTabCaption,
		PartIDScaleName, PartIDNumLabel, PartIDTernaryText:
		return true
	default:
		return false
	}
}

// String returns the canonical pylabview PARTID token (e.g. "NAME_LABEL"),
// or "Part(N)" when the value is outside the known vocabulary.
func (p PartID) String() string {
	if name, ok := partIDNames[p]; ok {
		return name
	}
	return fmt.Sprintf("Part(%d)", int32(p))
}

// partIDNames is the complete PARTID vocabulary mirrored from pylabview's
// LVparts.py. Two LabVIEW tokens share value 8015 (GRAPH_SCALE_LEGEND and
// TABLE); the canonical GRAPH_SCALE_LEGEND name is kept here.
var partIDNames = map[PartID]string{
	0:   "NO_PARTID",
	1:   "COSMETIC",
	2:   "INCREMENT",
	3:   "DECREMENT",
	4:   "LARGE_INCREMENT",
	5:   "LARGE_DECREMENT",
	6:   "PIXEL_INCREMENT",
	7:   "PIXEL_DECREMENT",
	8:   "HOUSING",
	9:   "FRAME",
	10:  "NUMERIC_TEXT",
	11:  "TEXT",
	12:  "RING_TEXT",
	13:  "SCROLLBAR",
	14:  "RING_PICTURE",
	15:  "RADIX",
	16:  "NAME_LABEL",
	17:  "SCALE",
	18:  "X_SCALE",
	19:  "Y_SCALE",
	20:  "OUT_OF_RANGE_BOX",
	21:  "BOOLEAN_BUTTON",
	22:  "BOOLEAN_TEXT",
	23:  "SLIDER_NEEDL_THUMB",
	24:  "SET_TO_DEFAULT",
	25:  "DECORATION",
	26:  "LIST_AREA",
	27:  "SCALE_MARKER",
	28:  "CONTENT_AREA",
	29:  "DDO_FRAME",
	30:  "INDEX_FRAME",
	31:  "FILL",
	32:  "GRAPH_LEGEND",
	33:  "GRAPH_PALETTE",
	34:  "X_FIT_BUTTON",
	35:  "Y_FIT_BUTTON",
	36:  "X_FIT_LOCK_BUTTON",
	37:  "Y_FIT_LOCK_BUTTON",
	38:  "X_SCROLLBAR",
	39:  "Y_SCROLLBAR",
	40:  "SCALE_TICK",
	41:  "COLOR_AREA",
	42:  "PALETTE_BACKGROUND",
	43:  "CONTRL_INDCTR_SYM",
	44:  "EXTRA_FRAME_PART",
	45:  "SCALE_MIN_TICK",
	46:  "PIX_MAP_PALETTE",
	47:  "SELECT_BUTTON",
	48:  "TEXT_BUTTON",
	49:  "ERASE_BUTTON",
	50:  "PEN_BUTTON",
	51:  "SUCKER_BUTTON",
	52:  "BUCKET_BUTTON",
	53:  "LINE_BUTTON",
	54:  "RECTANGLE_BUTTON",
	55:  "FILLED_RECT_BUTTON",
	56:  "OVAL_BUTTON",
	57:  "FILLED_OVAL_BUTTON",
	58:  "PATTERN",
	59:  "FOREGROUND_COLOR",
	60:  "BACKGROUND_COLOR",
	61:  "PIX_MAP_PAL_EXTRA",
	62:  "ZOOM_BAR",
	63:  "BOOLEAN_TRUE_LABEL",
	64:  "BOOLEAN_FALSE_LABEL",
	65:  "UNIT_LABEL",
	66:  "ANNEX",
	67:  "OLD_GRAPH_CURSOR",
	68:  "Z_SCALE",
	69:  "COLOR_RAMP",
	70:  "OUTPUT_INDICATOR",
	71:  "X_SCALE_UNIT_LABEL",
	72:  "Y_SCALE_UNIT_LABEL",
	73:  "Z_SCALE_UNIT_LABEL",
	74:  "GRAPH_MOVE_TOOL",
	75:  "GRAPH_ZOOM_TOOL",
	76:  "GRAPH_CURSOR_TOOL",
	77:  "GRAPH_X_FORMAT",
	78:  "GRAPH_Y_FORMAT",
	79:  "COMBO_BOX_BUTTON",
	80:  "DIAGRAM_IDENTIFIER",
	81:  "MENU_TITLE_LABEL",
	82:  "CAPTION",
	83:  "REFNUM_SYMBOL",
	84:  "KUNNAMED84",
	85:  "FORMERLY_ANNEX2",
	86:  "BOOLEAN_LIGHT",
	87:  "BOOLEAN_GLYPH",
	88:  "BOOLEAN_DIVOT",
	89:  "BOOLEAN_SHADOW",
	90:  "TAB",
	91:  "PAGE_LIST_BUTTON",
	92:  "TAB_CAPTION",
	93:  "TAB__BACKGROUND",
	94:  "SCALE_NAME",
	95:  "SLIDE_CAP",
	96:  "KUNNAMED96",
	97:  "CONTAINED_DATA_TYPE",
	98:  "POSITION_DATA_TYPE",
	99:  "TAB_GLYPH",
	100: "GRID",
	101: "NUM_LABEL",
	102: "SPLIT_BAR",
	103: "MUTLI_Y_SCROLLBAR",
	104: "GRAPH_VIEWPORT",
	105: "GRAB_HANDLE",
	106: "GRAPH_SPLITTER_BAR",
	107: "GRAPH_LEGEND_AREA",
	108: "GRAPH_LEGEND_SCRLBAR",
	109: "DATA_BINDING_STATUS",
	110: "TERNARY_TEXT",
	111: "TERNARY_BUTTON",
	112: "MULTISEG_PIPE_FLANGE",
	113: "MULTISEG_PIPE_ELBOW",
	114: "MULTISEG_PIPE_PIPE",
	115: "GRAPH_LEGEND_FRAME",
	116: "SCENE_GRAPH_DISPLAY",
	117: "OVERFLOW_STATUS",
	118: "RADIX_SHADOW",
	119: "CUSTOM_COSMETIC",
	120: "TYPEDEF_CORNER",

	8000: "NON_COLORABLE_DECAL",
	8001: "DIGITAL_DISPLAY",
	8002: "ARRAY_INDEX",
	8003: "VARIANT_INDEX",
	8004: "LISTBOX_DISPLAY",
	8005: "DATA_DISPLAY",
	8006: "MEASURE_DATA",
	8007: "KNOTUSED4",
	8008: "TREE_LEGEND",
	8009: "COLOR_RAMP_ARRAY",
	8010: "TYPE_DEFS_CONTROL",
	8011: "CURSOR_BUTTONS",
	8012: "HIGH_COLOR",
	8013: "LOW_COLOR",
	8014: "GRAPH_CURSOR",
	8015: "GRAPH_SCALE_LEGEND",
	8016: "IO_NAME_DISPLAY",
	8017: "TAB_CTRL_PAGE_SEL",
	8018: "BROWSE_BUTTON",
	8019: "GRAPH_PLOT_LEGEND",
}
