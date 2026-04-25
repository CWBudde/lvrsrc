package heap

import (
	"compress/zlib"
	"testing"
)

// FuzzDecodeEnvelope feeds arbitrary bytes to DecodeEnvelope. The contract
// is straightforward: the function must never panic, and either succeed
// or return an error. Any panic the fuzzer surfaces is a real bug.
func FuzzDecodeEnvelope(f *testing.F) {
	// Seed with a few well-formed and malformed payloads to give the
	// fuzzer a starting point that exercises both the outer header path
	// and the inflate path.
	f.Add(buildSeed(t(f), []byte{}))
	f.Add(buildSeed(t(f), []byte{0x10, 0x20, 0x30, 0x40}))
	f.Add(buildSeed(t(f), []byte("hello world tag stream contents")))
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0})              // 0-byte declared, no zlib body
	f.Add([]byte{0, 0, 0, 99, 0xFF, 0xFF}) // declared > available

	f.Fuzz(func(t *testing.T, payload []byte) {
		// Recover any panic into a test failure rather than a process crash.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("DecodeEnvelope panicked: %v", r)
			}
		}()
		_, _ = DecodeEnvelope(payload)
	})
}

// FuzzEncodeRoundTrip drives a fuzzed Content slice through
// EncodeEnvelope → DecodeEnvelope and asserts the round-trip preserves
// the bytes. Encode never sees pathological caller input here (we
// always set DeclaredSize / ContentLen consistently), so any decode
// failure on the re-encoded output is a real bug in either direction
// of the codec.
func FuzzEncodeRoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x01})
	f.Add([]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0})
	f.Add(make([]byte, 1024))

	f.Fuzz(func(t *testing.T, content []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Encode/Decode panicked: %v (content len %d)", r, len(content))
			}
		}()
		// Cap content to avoid pathologically large allocations from the
		// fuzzer; the corpus heaps top out at low thousands of bytes.
		if len(content) > 64*1024 {
			content = content[:64*1024]
		}
		env := Envelope{
			DeclaredSize: 4 + uint32(len(content)),
			ContentLen:   uint32(len(content)),
			Content:      content,
		}
		encoded, err := EncodeEnvelope(env)
		if err != nil {
			t.Fatalf("EncodeEnvelope: %v", err)
		}
		decoded, err := DecodeEnvelope(encoded)
		if err != nil {
			t.Fatalf("DecodeEnvelope of just-encoded: %v", err)
		}
		if decoded.ContentLen != env.ContentLen {
			t.Errorf("ContentLen drift: %d → %d", env.ContentLen, decoded.ContentLen)
		}
		if len(decoded.Content) != len(content) {
			t.Errorf("Content length drift: %d → %d", len(content), len(decoded.Content))
		}
		for i := range content {
			if decoded.Content[i] != content[i] {
				t.Errorf("Content byte drift at %d: %#x → %#x", i, content[i], decoded.Content[i])
				break
			}
		}
	})
}

// t adapts a *testing.F to the *testing.T-style helper buildSeed expects.
// buildEnvelopeBytes lives in envelope_test.go and signals errors via t.Fatalf;
// during seeding we want any internal failure to halt the corpus add, so we
// wrap testing.F.
func t(f *testing.F) *testing.T {
	// testing.F embeds testing.common but does not satisfy *testing.T.
	// Constructing a *testing.T at runtime isn't supported; the seed
	// helper below recreates the bytes inline instead.
	_ = f
	return &testing.T{}
}

// buildSeed builds a heap envelope payload synchronously without the
// helper from envelope_test.go (which needs a *testing.T to fail on).
// We accept the duplication for the sake of fuzz seeding; correctness
// is exercised by the deterministic tests already in this package.
func buildSeed(_ *testing.T, content []byte) []byte {
	inflated := make([]byte, 4+len(content))
	for i := 0; i < 4; i++ {
		inflated[i] = 0
	}
	inflated[0] = byte(len(content) >> 24)
	inflated[1] = byte(len(content) >> 16)
	inflated[2] = byte(len(content) >> 8)
	inflated[3] = byte(len(content))
	copy(inflated[4:], content)

	out, err := compressWithZlib(inflated)
	if err != nil {
		return nil
	}
	envelope := make([]byte, 4+len(out))
	envelope[0] = byte(len(inflated) >> 24)
	envelope[1] = byte(len(inflated) >> 16)
	envelope[2] = byte(len(inflated) >> 8)
	envelope[3] = byte(len(inflated))
	copy(envelope[4:], out)
	return envelope
}

func compressWithZlib(in []byte) ([]byte, error) {
	var buf zlibBuffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(in); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.b, nil
}

// zlibBuffer is a tiny io.Writer that captures written bytes without
// importing bytes from the test path twice.
type zlibBuffer struct {
	b []byte
}

func (z *zlibBuffer) Write(p []byte) (int, error) {
	z.b = append(z.b, p...)
	return len(p), nil
}
