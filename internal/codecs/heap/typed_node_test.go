package heap

import "testing"

func nodeWith(content []byte, sizeSpec byte) *Node {
	return &Node{Content: content, SizeSpec: sizeSpec}
}

func TestAsStdIntFullLengthSigned(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		want    int64
	}{
		{"1 byte positive", []byte{0x05}, 5},
		{"1 byte negative", []byte{0xFF}, -1},
		{"2 bytes BE", []byte{0xFF, 0xFE}, -2},
		{"4 bytes BE", []byte{0x00, 0x00, 0xCA, 0xFE}, 0xCAFE},
		{"4 bytes BE negative", []byte{0xFF, 0xFF, 0xFF, 0xFE}, -2},
		{"8 bytes BE", []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, 0x0001020304050607},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := nodeWith(tc.content, byte(len(tc.content)))
			got, err := n.AsStdInt(true)
			if err != nil {
				t.Fatalf("AsStdInt: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAsStdIntUnsigned(t *testing.T) {
	n := nodeWith([]byte{0xFF, 0xFE}, 2)
	got, err := n.AsStdInt(false)
	if err != nil {
		t.Fatalf("AsStdInt: %v", err)
	}
	if got != 0xFFFE {
		t.Errorf("AsStdInt(false) = %d, want %d", got, 0xFFFE)
	}
}

func TestAsStdIntRejectsBoolNode(t *testing.T) {
	n := nodeWith(nil, SizeSpecBoolFalse)
	if _, err := n.AsStdInt(true); err == nil {
		t.Error("AsStdInt on bool-false node returned nil error")
	}
	n = nodeWith(nil, SizeSpecBoolTrue)
	if _, err := n.AsStdInt(true); err == nil {
		t.Error("AsStdInt on bool-true node returned nil error")
	}
}

func TestAsStdIntRejectsExcessiveLength(t *testing.T) {
	// 9-byte content cannot fit a Go int64.
	n := nodeWith(make([]byte, 9), 9)
	if _, err := n.AsStdInt(true); err == nil {
		t.Error("AsStdInt on 9-byte content returned nil error")
	}
}

func TestAsTypeID(t *testing.T) {
	n := nodeWith([]byte{0x00, 0x05}, 2)
	got, err := n.AsTypeID()
	if err != nil {
		t.Fatalf("AsTypeID: %v", err)
	}
	if got != 5 {
		t.Errorf("AsTypeID = %d, want 5", got)
	}
}

func TestAsRect(t *testing.T) {
	// left=10, top=20, right=300, bottom=-1
	content := []byte{
		0x00, 0x0A, 0x00, 0x14, 0x01, 0x2C, 0xFF, 0xFF,
	}
	n := nodeWith(content, byte(len(content)))
	got, err := n.AsRect()
	if err != nil {
		t.Fatalf("AsRect: %v", err)
	}
	want := Rect{Left: 10, Top: 20, Right: 300, Bottom: -1}
	if got != want {
		t.Errorf("AsRect = %+v, want %+v", got, want)
	}
}

func TestAsRectRejectsWrongSize(t *testing.T) {
	for _, length := range []int{0, 4, 7, 9} {
		n := nodeWith(make([]byte, length), byte(length))
		if _, err := n.AsRect(); err == nil {
			t.Errorf("AsRect on %d-byte content returned nil error", length)
		}
	}
}

func TestAsPoint(t *testing.T) {
	// x=-5, y=42
	content := []byte{0xFF, 0xFB, 0x00, 0x2A}
	n := nodeWith(content, byte(len(content)))
	got, err := n.AsPoint()
	if err != nil {
		t.Fatalf("AsPoint: %v", err)
	}
	want := Point{X: -5, Y: 42}
	if got != want {
		t.Errorf("AsPoint = %+v, want %+v", got, want)
	}
}

func TestAsPointRejectsWrongSize(t *testing.T) {
	for _, length := range []int{0, 2, 3, 5, 8} {
		n := nodeWith(make([]byte, length), byte(length))
		if _, err := n.AsPoint(); err == nil {
			t.Errorf("AsPoint on %d-byte content returned nil error", length)
		}
	}
}

func TestAsString(t *testing.T) {
	n := nodeWith([]byte("Cluster"), 6) // SizeSpec value here is the on-disk var-len marker; content is what matters
	got, isNull, err := n.AsString()
	if err != nil {
		t.Fatalf("AsString: %v", err)
	}
	if isNull {
		t.Error("AsString isNull = true on non-empty content, want false")
	}
	if got != "Cluster" {
		t.Errorf("AsString = %q, want %q", got, "Cluster")
	}
}

func TestAsStringNullMarker(t *testing.T) {
	// SizeSpec 0 with no content represents a NULL string in pylabview's
	// HeapNodeString conventions (content == False).
	n := nodeWith(nil, SizeSpecBoolFalse)
	got, isNull, err := n.AsString()
	if err != nil {
		t.Fatalf("AsString: %v", err)
	}
	if !isNull {
		t.Errorf("AsString isNull = false on bool-false node, want true")
	}
	if got != "" {
		t.Errorf("AsString = %q, want empty", got)
	}
}

func TestAsStringEmptyContent(t *testing.T) {
	// SizeSpec 6 with zero-byte var-length content is a legitimate empty
	// string, not NULL.
	n := nodeWith([]byte{}, 6)
	got, isNull, err := n.AsString()
	if err != nil {
		t.Fatalf("AsString: %v", err)
	}
	if isNull {
		t.Errorf("empty-string node should not register as NULL")
	}
	if got != "" {
		t.Errorf("AsString = %q, want empty", got)
	}
}

func TestAsFloat32(t *testing.T) {
	// IEEE-754 big-endian: 1.0 = 0x3F800000
	n := nodeWith([]byte{0x3F, 0x80, 0x00, 0x00}, 4)
	got, err := n.AsFloat32()
	if err != nil {
		t.Fatalf("AsFloat32: %v", err)
	}
	if got != 1.0 {
		t.Errorf("AsFloat32 = %v, want 1.0", got)
	}
}

func TestAsFloat32WrongLengthRejected(t *testing.T) {
	for _, sz := range []int{0, 1, 2, 3, 5, 8} {
		n := nodeWith(make([]byte, sz), byte(sz))
		if _, err := n.AsFloat32(); err == nil {
			t.Errorf("AsFloat32(len=%d) returned nil error", sz)
		}
	}
}

func TestAsFloat64(t *testing.T) {
	// 2.0 in IEEE-754 BE: 0x4000000000000000
	n := nodeWith([]byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 8)
	got, err := n.AsFloat64()
	if err != nil {
		t.Fatalf("AsFloat64: %v", err)
	}
	if got != 2.0 {
		t.Errorf("AsFloat64 = %v, want 2.0", got)
	}
}

func TestAsFloat64WrongLengthRejected(t *testing.T) {
	for _, sz := range []int{0, 1, 4, 6, 7, 9, 16} {
		n := nodeWith(make([]byte, sz), byte(min(sz, 6)))
		if _, err := n.AsFloat64(); err == nil {
			t.Errorf("AsFloat64(len=%d) returned nil error", sz)
		}
	}
}
