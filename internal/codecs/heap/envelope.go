// Package heap implements the wire-level pieces shared by the FPHb
// (front-panel heap) and BDHb (block-diagram heap) resources. It is the
// foundation for Phase 9 and Phase 10 of the lvrsrc plan.
//
// At the wire level the two heaps are identical: a 4-byte big-endian
// declared-inflated size, followed by zlib-compressed bytes; once
// inflated, the buffer holds another 4-byte big-endian content length,
// followed by content_len bytes of tag-stream data.
//
//	+------------------------+    payload offset 0
//	|  DeclaredSize (u32 BE) |    = 4 + ContentLen (always)
//	+------------------------+    offset 4
//	|       zlib bytes       |
//	+------------------------+
//
//	After zlib inflation:
//	+------------------------+    inflated offset 0
//	|   ContentLen (u32 BE)  |
//	+------------------------+    offset 4
//	|        Content         |    ContentLen bytes of tag-stream
//	+------------------------+
//
// References: pylabview LVblock.py:5094-5179 (HeapVerb class) and
// LVblock.py:5103-5105 (BLOCK_CODING.ZLIB section coding). The same
// envelope is used by HeapVerb subclasses BDHb and FPHb (LVblock.py:5350-5362).
//
// Decode preserves the original compressed bytes alongside the inflated
// view so callers that have not edited Content can re-encode
// byte-for-byte. Encoding without those cached bytes (or after editing
// Content) recompresses with the standard library's zlib writer; output
// will not be byte-identical to LabVIEW's original then, but the
// envelope is round-trip-stable through Decode/Encode/Decode.
package heap

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// Envelope is the decoded heap envelope.
type Envelope struct {
	// DeclaredSize is the 4-byte BE field at offset 0 of the on-disk
	// payload. It records the inflated buffer length and is always
	// `4 + ContentLen` in valid payloads.
	DeclaredSize uint32

	// ContentLen is the 4-byte BE field at the start of the inflated
	// buffer. It records the byte count of the tag-stream that
	// follows.
	ContentLen uint32

	// Content is the inflated tag-stream bytes — opaque to the
	// envelope, semantically a sequence of HeapNode records that
	// Phase 9.2 / 9.3 will parse.
	Content []byte

	// Compressed is the original on-disk zlib byte-run (after the
	// outer DeclaredSize header). It is preserved so EncodeEnvelope
	// can reproduce the exact on-disk payload when Content has not
	// been mutated. Callers that build a fresh Envelope from scratch
	// should leave it nil; Encode will recompress automatically.
	Compressed []byte
}

const outerHeaderSize = 4

// DecodeEnvelope parses the on-disk payload into an Envelope.
//
// Validation: the outer DeclaredSize must equal the inflated length, and
// the inner ContentLen must equal `len(Content)`. Both checks fail
// loudly so a corrupt or truncated payload never silently produces a
// shorter Content slice.
func DecodeEnvelope(payload []byte) (Envelope, error) {
	if len(payload) < outerHeaderSize {
		return Envelope{}, fmt.Errorf("heap: payload too short for outer header: %d bytes (need at least %d)", len(payload), outerHeaderSize)
	}
	declaredSize := binary.BigEndian.Uint32(payload[:outerHeaderSize])
	compressed := append([]byte(nil), payload[outerHeaderSize:]...)
	inflated, err := inflate(compressed)
	if err != nil {
		return Envelope{}, fmt.Errorf("heap: inflate compressed payload: %w", err)
	}
	if uint32(len(inflated)) != declaredSize {
		return Envelope{}, fmt.Errorf("heap: declared size %d does not match inflated size %d", declaredSize, len(inflated))
	}
	if len(inflated) < 4 {
		return Envelope{}, fmt.Errorf("heap: inflated buffer too short for content length: %d bytes", len(inflated))
	}
	contentLen := binary.BigEndian.Uint32(inflated[:4])
	if uint32(len(inflated)-4) != contentLen {
		return Envelope{}, fmt.Errorf("heap: declared content length %d does not match available %d", contentLen, len(inflated)-4)
	}
	content := append([]byte(nil), inflated[4:]...)
	return Envelope{
		DeclaredSize: declaredSize,
		ContentLen:   contentLen,
		Content:      content,
		Compressed:   compressed,
	}, nil
}

// EncodeEnvelope serializes an Envelope back to its on-disk byte form.
//
// When Compressed is populated and inflates to the same bytes Content
// represents, those compressed bytes are reused verbatim — the result
// is byte-identical to the original DecodeEnvelope input. Otherwise the
// inflated buffer is rebuilt and recompressed with the standard
// library's zlib writer.
func EncodeEnvelope(env Envelope) ([]byte, error) {
	if env.ContentLen != uint32(len(env.Content)) {
		return nil, fmt.Errorf("heap: ContentLen = %d, want len(Content) %d", env.ContentLen, len(env.Content))
	}
	if env.DeclaredSize != 4+env.ContentLen {
		return nil, fmt.Errorf("heap: DeclaredSize = %d, want 4 + ContentLen %d", env.DeclaredSize, 4+env.ContentLen)
	}

	inflated := make([]byte, 4+len(env.Content))
	binary.BigEndian.PutUint32(inflated[:4], env.ContentLen)
	copy(inflated[4:], env.Content)

	compressed := env.Compressed
	if len(compressed) != 0 {
		// Verify the cache still matches Content; if not, fall through.
		if cached, err := inflate(compressed); err != nil || !bytes.Equal(cached, inflated) {
			compressed = nil
		}
	}
	if len(compressed) == 0 {
		var err error
		compressed, err = deflate(inflated)
		if err != nil {
			return nil, err
		}
	}

	out := make([]byte, outerHeaderSize+len(compressed))
	binary.BigEndian.PutUint32(out[:outerHeaderSize], env.DeclaredSize)
	copy(out[outerHeaderSize:], compressed)
	return out, nil
}

func inflate(compressed []byte) (_ []byte, err error) {
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := r.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	inflated, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return inflated, nil
}

func deflate(inflated []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(inflated); err != nil {
		writeErr := fmt.Errorf("heap: deflate payload: %w", err)
		if closeErr := w.Close(); closeErr != nil {
			return nil, fmt.Errorf("%v; additionally failed to close zlib writer: %w", writeErr, closeErr)
		}
		return nil, writeErr
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("heap: finalize deflate payload: %w", err)
	}
	return buf.Bytes(), nil
}
