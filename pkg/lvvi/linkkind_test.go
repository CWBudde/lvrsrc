package lvvi_test

import (
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

func TestLookupLinkKind(t *testing.T) {
	cases := map[string]lvvi.LinkKind{
		"TDCC": lvvi.LinkKindTypeDefToCCLink,
		"VILB": lvvi.LinkKindVIToLib,
		"IUVI": lvvi.LinkKindIUseToVILink,
		"VICC": lvvi.LinkKindVIToCCLink,
		"ZZZZ": lvvi.LinkKindUnknown,
	}
	for ident, want := range cases {
		if got := lvvi.LookupLinkKind(ident); got != want {
			t.Errorf("LookupLinkKind(%q) = %d, want %d", ident, got, want)
		}
	}
}

func TestLinkKindIdentAndDescription(t *testing.T) {
	if got := lvvi.LinkKindIdent(lvvi.LinkKindTypeDefToCCLink); got != "TDCC" {
		t.Errorf("ident = %q, want TDCC", got)
	}
	if got := lvvi.LinkKindIdent(lvvi.LinkKindUnknown); got != "" {
		t.Errorf("ident for Unknown = %q, want empty", got)
	}
	if got := lvvi.LinkKindDescription(lvvi.LinkKindVIToLib); got != "VI → Library" {
		t.Errorf("description = %q", got)
	}
}

// TestFrontPanelImports_TDCCFields exercises the typed projection of
// LIfp TDCC entries: every entry should resolve to LinkKindTypeDefToCCLink
// with a populated TypeID, KindDescription, and Offsets.
func TestFrontPanelImports_TDCCFields(t *testing.T) {
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), "reference-find-by-id.vi"), lvrsrc.OpenOptions{})
	if err != nil {
		t.Skipf("corpus fixture missing: %v", err)
	}
	m, _ := lvvi.DecodeKnownResources(f)
	deps, ok := m.FrontPanelImports()
	if !ok {
		t.Fatal("FrontPanelImports returned ok=false")
	}
	if len(deps) == 0 {
		t.Fatal("FrontPanelImports returned no entries")
	}
	for i, dep := range deps {
		if dep.LinkKind != lvvi.LinkKindTypeDefToCCLink {
			t.Errorf("entry %d LinkKind = %d, want LinkKindTypeDefToCCLink", i, dep.LinkKind)
		}
		if dep.LinkType != "TDCC" {
			t.Errorf("entry %d LinkType = %q, want TDCC", i, dep.LinkType)
		}
		if dep.KindDescription != "TypeDef → CustCtl" {
			t.Errorf("entry %d KindDescription = %q", i, dep.KindDescription)
		}
		if !dep.HasTypeID {
			t.Errorf("entry %d HasTypeID = false, want true", i)
		}
		if len(dep.Offsets) == 0 {
			t.Errorf("entry %d has no offsets", i)
		}
	}
}

// TestVIDependencies_TypedAndOpaque exercises the now-non-empty
// VIDependencies surface: entries that decoded into VILB targets should
// carry LinkKindVIToLib; VICC entries (no typed parser yet) should fall
// back to LinkKindVIToCCLink (still classified, just no extra fields).
func TestVIDependencies_TypedAndOpaque(t *testing.T) {
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), "is-float.vi"), lvrsrc.OpenOptions{})
	if err != nil {
		t.Skipf("corpus fixture missing: %v", err)
	}
	m, _ := lvvi.DecodeKnownResources(f)
	deps, ok := m.VIDependencies()
	if !ok {
		t.Fatal("VIDependencies returned ok=false")
	}
	if len(deps) != 2 {
		t.Fatalf("len(deps) = %d, want 2", len(deps))
	}
	if deps[0].LinkKind != lvvi.LinkKindVIToLib {
		t.Errorf("deps[0].LinkKind = %d, want LinkKindVIToLib", deps[0].LinkKind)
	}
	if deps[0].HasTypeID {
		t.Errorf("VILB entry should not carry TypeID")
	}
	if deps[1].LinkKind != lvvi.LinkKindVIToCCLink {
		t.Errorf("deps[1].LinkKind = %d, want LinkKindVIToCCLink", deps[1].LinkKind)
	}
	if deps[1].KindDescription != "VI → CustCtl" {
		t.Errorf("VICC description = %q", deps[1].KindDescription)
	}
}
