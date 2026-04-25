package heap

import "testing"

func TestReadU124(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want uint32
		used int
	}{
		{"single byte zero", []byte{0x00}, 0, 1},
		{"single byte 0x7E (sentinel boundary)", []byte{0x7E}, 0x7E, 1},
		{"single byte 0xFD", []byte{0xFD}, 0xFD, 1},
		{"escape FF + 16-bit", []byte{0xFF, 0x12, 0x34}, 0x1234, 3},
		{"escape FE + 32-bit", []byte{0xFE, 0xAA, 0xBB, 0xCC, 0xDD}, 0xAABBCCDD, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n, err := readU124(tc.in)
			if err != nil {
				t.Fatalf("readU124: %v", err)
			}
			if got != tc.want {
				t.Errorf("value = %#x, want %#x", got, tc.want)
			}
			if n != tc.used {
				t.Errorf("consumed = %d, want %d", n, tc.used)
			}
		})
	}
}

func TestReadU124RejectsTruncated(t *testing.T) {
	for _, in := range [][]byte{{}, {0xFF}, {0xFF, 0x01}, {0xFE}, {0xFE, 0x01, 0x02, 0x03}} {
		if _, _, err := readU124(in); err == nil {
			t.Errorf("readU124(%x) returned nil error", in)
		}
	}
}

func TestReadS124(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want int32
		used int
	}{
		{"positive small", []byte{0x05}, 5, 1},
		{"zero", []byte{0x00}, 0, 1},
		{"negative single byte", []byte{0xFF}, -1, 1},
		{"negative single -126", []byte{0x82}, -126, 1},
		{"escape -128 + 16-bit", []byte{0x80, 0x12, 0x34}, 0x1234, 3},
		{"escape -128 + 16-bit negative", []byte{0x80, 0xFF, 0xFE}, -2, 3},
		{"escape -127 + 32-bit", []byte{0x81, 0x00, 0xCA, 0xFE, 0xBE}, 0xCAFEBE, 5},
		{"escape -127 + 32-bit negative", []byte{0x81, 0xFF, 0xFF, 0xFF, 0xFE}, -2, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n, err := readS124(tc.in)
			if err != nil {
				t.Fatalf("readS124: %v", err)
			}
			if got != tc.want {
				t.Errorf("value = %d, want %d", got, tc.want)
			}
			if n != tc.used {
				t.Errorf("consumed = %d, want %d", n, tc.used)
			}
		})
	}
}

func TestReadS24(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want int32
		used int
	}{
		{"positive 16-bit", []byte{0x12, 0x34}, 0x1234, 2},
		{"negative 16-bit", []byte{0xFF, 0xFF}, -1, 2},
		{"escape 0x8000 + 32-bit", []byte{0x80, 0x00, 0x01, 0x00, 0x00, 0x00}, 0x01000000, 6},
		{"escape 0x8000 + 32-bit negative", []byte{0x80, 0x00, 0xFF, 0xFF, 0xFF, 0xFE}, -2, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n, err := readS24(tc.in)
			if err != nil {
				t.Fatalf("readS24: %v", err)
			}
			if got != tc.want {
				t.Errorf("value = %d, want %d", got, tc.want)
			}
			if n != tc.used {
				t.Errorf("consumed = %d, want %d", n, tc.used)
			}
		})
	}
}

func TestReadS24RejectsTruncated(t *testing.T) {
	for _, in := range [][]byte{nil, {0x00}, {0x80, 0x00, 0x01, 0x02, 0x03}} {
		if _, _, err := readS24(in); err == nil {
			t.Errorf("readS24(%x) returned nil error", in)
		}
	}
}
