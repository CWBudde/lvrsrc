package icon

import "testing"

func TestPalette16Length(t *testing.T) {
	if got := len(Palette16); got != 16 {
		t.Fatalf("len(Palette16) = %d, want 16", got)
	}
}

func TestPalette16BoundaryValues(t *testing.T) {
	// Port check: pylabview LABVIEW_COLOR_PALETTE_16 begins with white (0xFFFFFF)
	// and ends with black (0x000000). Palette entries are packed ARGB with alpha
	// pinned to 0xFF.
	cases := []struct {
		idx  int
		want uint32
	}{
		{0, 0xFFFFFFFF},
		{15, 0xFF000000},
	}
	for _, tc := range cases {
		if got := Palette16[tc.idx]; got != tc.want {
			t.Errorf("Palette16[%d] = %#08x, want %#08x", tc.idx, got, tc.want)
		}
	}
}

func TestPalette16SpotValues(t *testing.T) {
	// Spot-check a handful of mid-palette entries to ensure the port did not
	// truncate or reorder. pylabview LVmisc.py:88-91.
	cases := []struct {
		idx  int
		want uint32
	}{
		{1, 0xFFFFFF00},  // yellow
		{3, 0xFFFF0000},  // red
		{6, 0xFF0000FF},  // blue
		{8, 0xFF00FF00},  // green
		{12, 0xFFC0C0C0}, // silver
	}
	for _, tc := range cases {
		if got := Palette16[tc.idx]; got != tc.want {
			t.Errorf("Palette16[%d] = %#08x, want %#08x", tc.idx, got, tc.want)
		}
	}
}

func TestPalette256Length(t *testing.T) {
	if got := len(Palette256); got != 256 {
		t.Fatalf("len(Palette256) = %d, want 256", got)
	}
}

func TestPalette256BoundaryValues(t *testing.T) {
	// pylabview LVmisc.py:52-85. First entry is a near-white cell (0xF1F1F1);
	// last entry is pure black.
	cases := []struct {
		idx  int
		want uint32
	}{
		{0, 0xFFF1F1F1},
		{255, 0xFF000000},
	}
	for _, tc := range cases {
		if got := Palette256[tc.idx]; got != tc.want {
			t.Errorf("Palette256[%d] = %#08x, want %#08x", tc.idx, got, tc.want)
		}
	}
}

func TestPalette256SpotValues(t *testing.T) {
	// Spot-check specific entries identifiable by their position in the
	// systematic FF/CC/99/66/33/00 colour-cube pattern used by pylabview.
	cases := []struct {
		idx  int
		want uint32
	}{
		{5, 0xFFFFFF00},   // row 0 col 5: yellow
		{35, 0xFFFF0000},  // row 4 col 3: pure red
		{43, 0xFFCCCCCC},  // row 5 col 3: light grey
		{215, 0xFFEE0000}, // dark-red ramp
		{245, 0xFFEEEEEE}, // grey ramp
	}
	for _, tc := range cases {
		if got := Palette256[tc.idx]; got != tc.want {
			t.Errorf("Palette256[%d] = %#08x, want %#08x", tc.idx, got, tc.want)
		}
	}
}
