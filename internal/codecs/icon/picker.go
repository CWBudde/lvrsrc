package icon

import "github.com/CWBudde/lvrsrc/internal/codecs"

// PayloadLookup returns the raw bytes of a resource identified by FourCC, or
// (nil, false) when the file does not carry that resource. It exists so the
// picker stays decoupled from any particular file/container type.
type PayloadLookup func(fourCC string) ([]byte, bool)

// PickResult bundles the selected icon variant with its decoded Value.
type PickResult struct {
	FourCC string
	Value  Value
}

// PickBest tries the three icon variants in descending colour-depth order
// (icl8 → icl4 → ICON) and returns the first one whose payload is present,
// the right size, and decodes cleanly. Returns ok = false when none of the
// three is available.
//
// Out-of-spec payloads (wrong size, decode error) are silently skipped so a
// damaged colour icon never stops the fallback to the mono ICON.
func PickBest(ctx codecs.Context, lookup PayloadLookup) (PickResult, bool) {
	candidates := []struct {
		fourCC codecs.FourCC
		codec  codecs.ResourceCodec
		size   int
	}{
		{Color8FourCC, Color8Codec{}, color8Spec.rawSize},
		{Color4FourCC, Color4Codec{}, color4Spec.rawSize},
		{MonoFourCC, MonoCodec{}, monoSpec.rawSize},
	}

	for _, c := range candidates {
		payload, ok := lookup(string(c.fourCC))
		if !ok || len(payload) != c.size {
			continue
		}
		raw, err := c.codec.Decode(ctx, payload)
		if err != nil {
			continue
		}
		v, ok := raw.(Value)
		if !ok {
			continue
		}
		return PickResult{FourCC: string(c.fourCC), Value: v}, true
	}
	return PickResult{}, false
}
