package binaryx

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"
)

type fixedWriterAt struct {
	buf []byte
}

func newFixedWriterAt(size int) *fixedWriterAt {
	return &fixedWriterAt{buf: make([]byte, size)}
}

func (w *fixedWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset %d", off)
	}
	if off > int64(len(w.buf)) || int64(len(p)) > int64(len(w.buf))-off {
		return 0, io.ErrShortWrite
	}

	return copy(w.buf[off:], p), nil
}

func TestWriterNumbersBigEndian(t *testing.T) {
	dst := newFixedWriterAt(14)
	w := NewWriter(dst, binary.BigEndian)

	if err := w.WriteU16(0, 0x1234); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteU32(2, 0x01234567); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteU64(6, 0x0123456789abcdef); err != nil {
		t.Fatal(err)
	}

	want := []byte{
		0x12, 0x34,
		0x01, 0x23, 0x45, 0x67,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
	}
	if string(dst.buf) != string(want) {
		t.Fatalf("got %x want %x", dst.buf, want)
	}
}

func TestWriterNumbersLittleEndian(t *testing.T) {
	dst := newFixedWriterAt(6)
	w := NewWriter(dst, binary.LittleEndian)

	if err := w.WriteU16(0, 0x1234); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteU32(2, 0x12345678); err != nil {
		t.Fatal(err)
	}

	want := []byte{0x34, 0x12, 0x78, 0x56, 0x34, 0x12}
	if string(dst.buf) != string(want) {
		t.Fatalf("got %x want %x", dst.buf, want)
	}
}

func TestWriterBytes(t *testing.T) {
	dst := newFixedWriterAt(6)
	w := NewWriter(dst, binary.BigEndian)

	if err := w.WriteBytes(1, []byte("xyz")); err != nil {
		t.Fatal(err)
	}

	if got := string(dst.buf); got != "\x00xyz\x00\x00" {
		t.Fatalf("got %q", got)
	}
}

func TestWriterPascalString(t *testing.T) {
	dst := newFixedWriterAt(8)
	w := NewWriter(dst, binary.BigEndian)

	n, err := w.WritePascalString(0, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Fatalf("got %d", n)
	}

	n, err = w.WritePascalString(6, "x")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("got %d", n)
	}

	want := []byte{5, 'h', 'e', 'l', 'l', 'o', 1, 'x'}
	if string(dst.buf) != string(want) {
		t.Fatalf("got %x want %x", dst.buf, want)
	}
}

func TestWriterPascalStringTooLong(t *testing.T) {
	dst := newFixedWriterAt(260)
	w := NewWriter(dst, binary.BigEndian)

	if _, err := w.WritePascalString(0, strings.Repeat("a", 256)); err == nil {
		t.Fatal("expected length error")
	}
}

func TestWriterPatchU32(t *testing.T) {
	dst := newFixedWriterAt(8)
	w := NewWriter(dst, binary.BigEndian)

	patch, err := w.PlaceholderU32(2)
	if err != nil {
		t.Fatal(err)
	}
	if got := dst.buf; string(got) != "\x00\x00\x00\x00\x00\x00\x00\x00" {
		t.Fatalf("got %x", got)
	}

	if err := patch(0x89abcdef); err != nil {
		t.Fatal(err)
	}

	want := []byte{0x00, 0x00, 0x89, 0xab, 0xcd, 0xef, 0x00, 0x00}
	if string(dst.buf) != string(want) {
		t.Fatalf("got %x want %x", dst.buf, want)
	}
}

func TestWriterBounds(t *testing.T) {
	dst := newFixedWriterAt(3)
	w := NewWriter(dst, binary.BigEndian)

	if err := w.WriteU32(0, 1); err == nil {
		t.Fatal("expected bounds error")
	}
	if err := w.WriteBytes(-1, []byte{1}); err == nil {
		t.Fatal("expected offset error")
	}
	if _, err := w.WritePascalString(3, "x"); err == nil {
		t.Fatal("expected bounds error")
	}
	if _, err := w.PlaceholderU32(1); err == nil {
		t.Fatal("expected bounds error")
	}
}
