package binaryx

import (
	"encoding/binary"
	"testing"
)

func TestReaderNumbersBigEndian(t *testing.T) {
	r := NewReader([]byte{
		0x7f,
		0x12, 0x34,
		0x01, 0x23, 0x45, 0x67,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
	}, binary.BigEndian)

	u8, err := r.U8(0)
	if err != nil || u8 != 0x7f {
		t.Fatalf("U8 got (%x, %v)", u8, err)
	}

	u16, err := r.U16(1)
	if err != nil || u16 != 0x1234 {
		t.Fatalf("U16 got (%x, %v)", u16, err)
	}

	u32, err := r.U32(3)
	if err != nil || u32 != 0x01234567 {
		t.Fatalf("U32 got (%x, %v)", u32, err)
	}

	u64, err := r.U64(7)
	if err != nil || u64 != 0x0123456789abcdef {
		t.Fatalf("U64 got (%x, %v)", u64, err)
	}
}

func TestReaderNumbersLittleEndian(t *testing.T) {
	r := NewReader([]byte{0x34, 0x12, 0x78, 0x56, 0x34, 0x12}, binary.LittleEndian)

	u16, err := r.U16(0)
	if err != nil || u16 != 0x1234 {
		t.Fatalf("U16 got (%x, %v)", u16, err)
	}

	u32, err := r.U32(2)
	if err != nil || u32 != 0x12345678 {
		t.Fatalf("U32 got (%x, %v)", u32, err)
	}
}

func TestReaderBytes(t *testing.T) {
	r := NewReader([]byte("abcdef"), binary.BigEndian)
	b, err := r.Bytes(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "cde" {
		t.Fatalf("got %q", string(b))
	}
}

func TestReaderPascalString(t *testing.T) {
	r := NewReader([]byte{5, 'h', 'e', 'l', 'l', 'o', 1, 'x'}, binary.BigEndian)
	s, n, err := r.PascalString(0)
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" || n != 6 {
		t.Fatalf("got (%q, %d)", s, n)
	}

	s, n, err = r.PascalString(6)
	if err != nil {
		t.Fatal(err)
	}
	if s != "x" || n != 2 {
		t.Fatalf("got (%q, %d)", s, n)
	}
}

func TestReaderCString(t *testing.T) {
	r := NewReader([]byte{'a', 'b', 0, 'z'}, binary.BigEndian)
	s, n, err := r.CString(0)
	if err != nil {
		t.Fatal(err)
	}
	if s != "ab" || n != 3 {
		t.Fatalf("got (%q, %d)", s, n)
	}
}

func TestReaderBounds(t *testing.T) {
	r := NewReader([]byte{1, 2, 3}, binary.BigEndian)

	if _, err := r.U64(0); err == nil {
		t.Fatal("expected bounds error")
	}
	if _, _, err := r.PascalString(3); err == nil {
		t.Fatal("expected bounds error")
	}
	if _, _, err := r.CString(2); err == nil {
		t.Fatal("expected missing NUL error")
	}
}
