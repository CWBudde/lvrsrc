package rsrcwire

import "testing"

// Placeholder fuzz target for CI wiring in Phase 1.
func FuzzParseFile(f *testing.F) {
	f.Add([]byte("RSRC"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = data
	})
}
