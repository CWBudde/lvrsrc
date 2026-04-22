package binaryx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Reader provides offset-based primitive reads with bounds checks.
type Reader struct {
	ra    io.ReaderAt
	size  int64
	order binary.ByteOrder
}

// NewReader creates a Reader over b using order.
func NewReader(b []byte, order binary.ByteOrder) *Reader {
	if order == nil {
		order = binary.BigEndian
	}

	return &Reader{
		ra:    bytes.NewReader(b),
		size:  int64(len(b)),
		order: order,
	}
}

func (r *Reader) readAt(off int64, n int64) ([]byte, error) {
	if off < 0 {
		return nil, fmt.Errorf("read at offset %d: negative offset", off)
	}
	if n < 0 {
		return nil, fmt.Errorf("read at offset %d: negative size %d", off, n)
	}
	if off+n > r.size {
		return nil, fmt.Errorf("read at offset %d size %d: out of bounds (size=%d)", off, n, r.size)
	}

	buf := make([]byte, n)
	if _, err := r.ra.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("read at offset %d size %d: %w", off, n, err)
	}

	return buf, nil
}

func (r *Reader) U8(off int64) (uint8, error) {
	b, err := r.readAt(off, 1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *Reader) U16(off int64) (uint16, error) {
	b, err := r.readAt(off, 2)
	if err != nil {
		return 0, err
	}
	return r.order.Uint16(b), nil
}

func (r *Reader) U32(off int64) (uint32, error) {
	b, err := r.readAt(off, 4)
	if err != nil {
		return 0, err
	}
	return r.order.Uint32(b), nil
}

func (r *Reader) U64(off int64) (uint64, error) {
	b, err := r.readAt(off, 8)
	if err != nil {
		return 0, err
	}
	return r.order.Uint64(b), nil
}

func (r *Reader) Bytes(off int64, n int) ([]byte, error) {
	b, err := r.readAt(off, int64(n))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// PascalString reads a one-byte length followed by bytes.
// It returns the decoded string and consumed byte count.
func (r *Reader) PascalString(off int64) (string, int64, error) {
	ln, err := r.U8(off)
	if err != nil {
		return "", 0, err
	}
	if ln == 0 {
		return "", 1, nil
	}

	b, err := r.readAt(off+1, int64(ln))
	if err != nil {
		return "", 0, err
	}

	return string(b), int64(1 + ln), nil
}

// CString reads bytes until a NUL terminator.
// It returns the decoded string and consumed byte count including terminator.
func (r *Reader) CString(off int64) (string, int64, error) {
	if off < 0 || off >= r.size {
		return "", 0, fmt.Errorf("c-string at offset %d: out of bounds (size=%d)", off, r.size)
	}

	var buf []byte
	for i := off; i < r.size; i++ {
		b, err := r.readAt(i, 1)
		if err != nil {
			return "", 0, err
		}
		if b[0] == 0 {
			return string(buf), int64(len(buf) + 1), nil
		}
		buf = append(buf, b[0])
	}

	return "", 0, fmt.Errorf("c-string at offset %d: missing NUL terminator", off)
}
