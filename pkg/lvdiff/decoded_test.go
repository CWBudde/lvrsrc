package lvdiff

import (
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// recordingCodec captures every Context passed to Decode so tests can assert
// that pkg/lvdiff threads per-file context through the decoded-diff pipeline.
type recordingCodec struct {
	fourCC string
	calls  *[]codecs.Context
}

func (r recordingCodec) Capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        r.fourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier1,
	}
}

func (r recordingCodec) Decode(ctx codecs.Context, payload []byte) (any, error) {
	*r.calls = append(*r.calls, ctx)
	return append([]byte(nil), payload...), nil
}

func (r recordingCodec) Encode(ctx codecs.Context, value any) ([]byte, error) {
	return nil, nil
}

func (r recordingCodec) Validate(ctx codecs.Context, payload []byte) []validate.Issue {
	return nil
}

func TestContextFromFileDerivation(t *testing.T) {
	if got := contextFromFile(nil); got != (codecs.Context{}) {
		t.Fatalf("contextFromFile(nil) = %+v, want zero value", got)
	}

	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 0x1234},
		Kind:   lvrsrc.FileKindControl,
	}
	got := contextFromFile(f)
	want := codecs.Context{FileVersion: 0x1234, Kind: lvrsrc.FileKindControl}
	if got != want {
		t.Fatalf("contextFromFile(%+v) = %+v, want %+v", f, got, want)
	}
}

func TestMakeCodecDifferUsesAContextForOldAndBContextForNew(t *testing.T) {
	var calls []codecs.Context
	codec := recordingCodec{fourCC: "TEST", calls: &calls}

	aCtx := codecs.Context{FileVersion: 0x0A0A, Kind: lvrsrc.FileKindVI}
	bCtx := codecs.Context{FileVersion: 0x0B0B, Kind: lvrsrc.FileKindTemplate}

	differ := makeCodecDiffer(codec, aCtx, bCtx)
	differ("TEST", 0, []byte{0x01}, []byte{0x02})

	if len(calls) != 2 {
		t.Fatalf("Decode calls = %d, want 2; got %+v", len(calls), calls)
	}
	if calls[0] != aCtx {
		t.Fatalf("old-payload Decode got ctx = %+v, want aCtx %+v", calls[0], aCtx)
	}
	if calls[1] != bCtx {
		t.Fatalf("new-payload Decode got ctx = %+v, want bCtx %+v", calls[1], bCtx)
	}
}

func TestDefaultDecodedDiffersDeriveContextFromFiles(t *testing.T) {
	a := baseFile()
	a.Header.FormatVersion = 0x0003
	a.Kind = lvrsrc.FileKindVI

	b := baseFile()
	b.Header.FormatVersion = 0x0009
	b.Kind = lvrsrc.FileKindTemplate

	aCtxWant := codecs.Context{FileVersion: a.Header.FormatVersion, Kind: a.Kind}
	bCtxWant := codecs.Context{FileVersion: b.Header.FormatVersion, Kind: b.Kind}

	if got := contextFromFile(a); got != aCtxWant {
		t.Fatalf("contextFromFile(a) = %+v, want %+v", got, aCtxWant)
	}
	if got := contextFromFile(b); got != bCtxWant {
		t.Fatalf("contextFromFile(b) = %+v, want %+v", got, bCtxWant)
	}

	diffs := defaultDecodedDiffers(a, b)
	if len(diffs) == 0 {
		t.Fatal("defaultDecodedDiffers returned empty map; expected shipped codecs")
	}
	// Sanity: the map is keyed by FourCC of registered real codecs; we do
	// not need to invoke them here because TestMakeCodecDifferUses... and
	// TestContextFromFileDerivation already cover the plumbing.
	for _, fourCC := range []string{"CONP", "vers", "STRG"} {
		if _, ok := diffs[fourCC]; !ok {
			t.Errorf("expected default differ for %q, missing", fourCC)
		}
	}
}
