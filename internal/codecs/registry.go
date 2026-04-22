package codecs

import (
	"fmt"
	"sort"
)

// Registry maps FourCC block types to their ResourceCodec. The zero value is
// not usable; always construct with New.
//
// Registries are intended to be built at startup and then read concurrently.
// No internal locking is provided; concurrent Register calls are not safe.
type Registry struct {
	codecs map[FourCC]ResourceCodec
}

// New returns an empty Registry ready for Register calls.
func New() *Registry {
	return &Registry{codecs: make(map[FourCC]ResourceCodec)}
}

// Register adds codec keyed by codec.Capability().FourCC. It panics if the
// FourCC is empty or already registered, since either indicates a programmer
// error in registration wiring.
func (r *Registry) Register(codec ResourceCodec) {
	c := codec.Capability()
	if c.FourCC == "" {
		panic("codecs: Register called with empty FourCC")
	}
	if _, exists := r.codecs[c.FourCC]; exists {
		panic(fmt.Sprintf("codecs: duplicate registration for FourCC %q", c.FourCC))
	}
	r.codecs[c.FourCC] = codec
}

// Lookup returns the codec registered for fourCC, or OpaqueCodec{} if none is
// registered. It never returns nil.
func (r *Registry) Lookup(fourCC FourCC) ResourceCodec {
	if c, ok := r.codecs[fourCC]; ok {
		return c
	}
	return OpaqueCodec{}
}

// Has reports whether a non-opaque codec is registered for fourCC.
func (r *Registry) Has(fourCC FourCC) bool {
	_, ok := r.codecs[fourCC]
	return ok
}

// Capabilities returns a snapshot of all registered capabilities, sorted by
// FourCC for stable iteration order.
func (r *Registry) Capabilities() []Capability {
	out := make([]Capability, 0, len(r.codecs))
	for _, c := range r.codecs {
		out = append(out, c.Capability())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FourCC < out[j].FourCC })
	return out
}
