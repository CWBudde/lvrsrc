package lvmeta

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/codecs/vers"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// Mutator configures Tier 2 metadata edits. Additional behavior will be added
// as the metadata editing API grows in Phase 4.4+.
type Mutator struct {
	// Strict switches the post-edit safety gate to fail on new codec
	// warnings in addition to always failing on codec errors.
	Strict bool

	// registry lets tests inject a custom codec registry. Zero-value
	// Mutators fall back to the package-level defaultRegistry populated
	// from pkg/lvmeta/lvmeta.go's init wiring.
	registry *codecs.Registry
}

func (m Mutator) effectiveRegistry() *codecs.Registry {
	if m.registry != nil {
		return m.registry
	}
	return defaultRegistry
}

var defaultRegistry = newDefaultRegistry()

func newDefaultRegistry() *codecs.Registry {
	r := codecs.New()
	r.Register(strg.Codec{})
	r.Register(vers.Codec{})
	return r
}

func contextFromFile(f *lvrsrc.File) codecs.Context {
	if f == nil {
		return codecs.Context{}
	}
	return codecs.Context{
		FileVersion: f.Header.FormatVersion,
		Kind:        f.Kind,
	}
}

type sectionRef struct {
	BlockIndex   int
	SectionIndex int
}

func findSectionsByType(f *lvrsrc.File, fourCC string) []sectionRef {
	if f == nil {
		return nil
	}

	var refs []sectionRef
	for bi, block := range f.Blocks {
		if block.Type != fourCC {
			continue
		}
		for si := range block.Sections {
			refs = append(refs, sectionRef{
				BlockIndex:   bi,
				SectionIndex: si,
			})
		}
	}
	return refs
}

func requireSingleSectionByType(f *lvrsrc.File, fourCC string) (sectionRef, bool, error) {
	refs := findSectionsByType(f, fourCC)
	switch len(refs) {
	case 0:
		return sectionRef{}, false, nil
	case 1:
		return refs[0], true, nil
	default:
		return sectionRef{}, false, fmt.Errorf("lvmeta: found %d sections for %q, want exactly 1", len(refs), fourCC)
	}
}
