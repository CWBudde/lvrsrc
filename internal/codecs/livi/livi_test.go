package livi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestCapability(t *testing.T) {
	c := Codec{}.Capability()
	if c.FourCC != FourCC {
		t.Fatalf("FourCC = %q, want %q", c.FourCC, FourCC)
	}
	if c.Safety != codecs.SafetyTier1 {
		t.Fatalf("Safety = %v, want SafetyTier1", c.Safety)
	}
}

func TestKnownMarkers(t *testing.T) {
	for _, m := range []string{"LVIN", "LVCC", "LVIT", "LLBV"} {
		if !isKnownMarker(m) {
			t.Errorf("isKnownMarker(%q) = false", m)
		}
	}
	if isKnownMarker("XYZW") {
		t.Error("isKnownMarker(\"XYZW\") = true, want false")
	}
}

func TestDecodeAcceptsLVCCMarkerForCTLFiles(t *testing.T) {
	// Empty LIvi payload with LVCC marker (.ctl file): version + marker + count=0 + footer.
	payload := []byte{
		0, 1, // version
		'L', 'V', 'C', 'C',
		0, 0, 0, 0, // count = 0
		0, 3, // footer
	}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if val.Marker != "LVCC" {
		t.Errorf("Marker = %q, want %q", val.Marker, "LVCC")
	}
	if val.EntryCount != 0 || len(val.Body) != 0 {
		t.Errorf("Body = %+v, want empty", val.Body)
	}
}

func TestValidateWarnsOnUnknownMarker(t *testing.T) {
	payload := []byte{0, 1, 'X', 'Y', 'Z', 'W', 0, 0, 0, 0, 0, 3}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	if len(issues) == 0 {
		t.Fatal("Validate(unknown marker) returned no issues")
	}
	if issues[0].Severity != validate.SeverityWarning {
		t.Errorf("severity = %v, want warning", issues[0].Severity)
	}
}

func TestCorpusRoundTrip(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	total := 0
	markerCounts := map[string]int{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("open %s: %v", e.Name(), err)
		}
		for _, block := range f.Blocks {
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				total++
				v, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("%s id=%d Decode: %v", e.Name(), section.Index, err)
				}
				markerCounts[v.(Value).Marker]++
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("%s id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s id=%d round-trip mismatch", e.Name(), section.Index)
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no LIvi sections in corpus")
	}
	t.Logf("exercised %d LIvi section(s); markers seen: %v", total, markerCounts)
}

// TestEncodeRejectsBadInput exercises the two error branches of Encode
// (nil typed pointer + wrong concrete type) that the round-trip fixtures
// never hit.
func TestEncodeRejectsBadInput(t *testing.T) {
	if _, err := (Codec{}).Encode(codecs.Context{}, (*Value)(nil)); err == nil {
		t.Errorf("Encode(nil *Value) returned no error")
	}
	if _, err := (Codec{}).Encode(codecs.Context{}, "not a Value"); err == nil {
		t.Errorf("Encode(string) returned no error")
	}
}
