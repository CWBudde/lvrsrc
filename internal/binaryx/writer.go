package binaryx

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Writer provides offset-based primitive writes with contextual errors.
type Writer struct {
	wa    io.WriterAt
	order binary.ByteOrder
}

// NewWriter creates a Writer over wa using order.
func NewWriter(wa io.WriterAt, order binary.ByteOrder) *Writer {
	if order == nil {
		order = binary.BigEndian
	}

	return &Writer{
		wa:    wa,
		order: order,
	}
}

func (w *Writer) writeAt(off int64, b []byte) error {
	if off < 0 {
		return fmt.Errorf("write at offset %d: negative offset", off)
	}

	n, err := w.wa.WriteAt(b, off)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("write at offset %d size %d: %w", off, len(b), err)
	}

	return nil
}

func (w *Writer) WriteU16(off int64, v uint16) error {
	b := make([]byte, 2)
	w.order.PutUint16(b, v)
	return w.writeAt(off, b)
}

func (w *Writer) WriteU32(off int64, v uint32) error {
	b := make([]byte, 4)
	w.order.PutUint32(b, v)
	return w.writeAt(off, b)
}

func (w *Writer) WriteU64(off int64, v uint64) error {
	b := make([]byte, 8)
	w.order.PutUint64(b, v)
	return w.writeAt(off, b)
}

func (w *Writer) WriteBytes(off int64, data []byte) error {
	return w.writeAt(off, data)
}

// WritePascalString writes a one-byte length followed by the string bytes.
// It returns the consumed byte count.
func (w *Writer) WritePascalString(off int64, s string) (int64, error) {
	if len(s) > 255 {
		return 0, fmt.Errorf("pascal string at offset %d: length %d exceeds 255", off, len(s))
	}

	buf := make([]byte, 1+len(s))
	buf[0] = byte(len(s))
	copy(buf[1:], s)
	if err := w.writeAt(off, buf); err != nil {
		return 0, err
	}

	return int64(len(buf)), nil
}

// PlaceholderU32 writes a zero placeholder and returns a patch function for it.
func (w *Writer) PlaceholderU32(off int64) (func(uint32) error, error) {
	if err := w.WriteU32(off, 0); err != nil {
		return nil, err
	}

	return func(v uint32) error {
		return w.WriteU32(off, v)
	}, nil
}
