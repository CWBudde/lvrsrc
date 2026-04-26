package lvmeta

import (
	"errors"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestNewDefaultRegistryIncludesShippedTier2Codecs(t *testing.T) {
	r := newDefaultRegistry()

	cases := []struct {
		fourCC string
	}{
		{fourCC: "STRG"},
		{fourCC: "vers"},
	}
	for _, tc := range cases {
		t.Run(tc.fourCC, func(t *testing.T) {
			if !r.Has(tc.fourCC) {
				t.Fatalf("Has(%q) = false, want true", tc.fourCC)
			}
			c := r.Lookup(tc.fourCC).Capability()
			if c.FourCC != tc.fourCC {
				t.Fatalf("Lookup(%q).Capability().FourCC = %q, want %q", tc.fourCC, c.FourCC, tc.fourCC)
			}
			if c.Safety != codecs.SafetyTier2 {
				t.Fatalf("Lookup(%q).Capability().Safety = %v, want SafetyTier2", tc.fourCC, c.Safety)
			}
		})
	}
}

func TestContextFromFile(t *testing.T) {
	t.Run("nil file", func(t *testing.T) {
		if got := contextFromFile(nil); got != (codecs.Context{}) {
			t.Fatalf("contextFromFile(nil) = %+v, want zero", got)
		}
	})

	t.Run("propagates header version and file kind", func(t *testing.T) {
		f := &lvrsrc.File{
			Header: lvrsrc.Header{FormatVersion: 3},
			Kind:   lvrsrc.FileKindControl,
		}

		got := contextFromFile(f)
		if got.FileVersion != 3 {
			t.Fatalf("FileVersion = %d, want 3", got.FileVersion)
		}
		if got.Kind != lvrsrc.FileKindControl {
			t.Fatalf("Kind = %q, want %q", got.Kind, lvrsrc.FileKindControl)
		}
	})
}

func TestFindSectionsByTypeReturnsDeterministicOrder(t *testing.T) {
	f := &lvrsrc.File{
		Blocks: []lvrsrc.Block{
			{
				Type: "vers",
				Sections: []lvrsrc.Section{
					{Index: 7},
				},
			},
			{
				Type: "STRG",
				Sections: []lvrsrc.Section{
					{Index: 10},
					{Index: 11},
				},
			},
			{
				Type: "STRG",
				Sections: []lvrsrc.Section{
					{Index: 12},
				},
			},
		},
	}

	got := findSectionsByType(f, "STRG")
	if len(got) != 3 {
		t.Fatalf("len(findSectionsByType(STRG)) = %d, want 3", len(got))
	}

	want := []sectionRef{
		{BlockIndex: 1, SectionIndex: 0},
		{BlockIndex: 1, SectionIndex: 1},
		{BlockIndex: 2, SectionIndex: 0},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("findSectionsByType()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRequireSingleSectionByType(t *testing.T) {
	t.Run("zero matches", func(t *testing.T) {
		ref, ok, err := requireSingleSectionByType(&lvrsrc.File{}, "STRG")
		if err != nil {
			t.Fatalf("requireSingleSectionByType zero-match error = %v, want nil", err)
		}
		if ok {
			t.Fatalf("requireSingleSectionByType zero-match ok = true, want false")
		}
		if ref != (sectionRef{}) {
			t.Fatalf("requireSingleSectionByType zero-match ref = %+v, want zero", ref)
		}
	})

	t.Run("one match", func(t *testing.T) {
		f := &lvrsrc.File{
			Blocks: []lvrsrc.Block{
				{
					Type: "STRG",
					Sections: []lvrsrc.Section{
						{Index: 99},
					},
				},
			},
		}

		ref, ok, err := requireSingleSectionByType(f, "STRG")
		if err != nil {
			t.Fatalf("requireSingleSectionByType one-match error = %v, want nil", err)
		}
		if !ok {
			t.Fatalf("requireSingleSectionByType one-match ok = false, want true")
		}
		if ref != (sectionRef{BlockIndex: 0, SectionIndex: 0}) {
			t.Fatalf("requireSingleSectionByType one-match ref = %+v, want {0 0}", ref)
		}
	})

	t.Run("many matches", func(t *testing.T) {
		f := &lvrsrc.File{
			Blocks: []lvrsrc.Block{
				{
					Type: "STRG",
					Sections: []lvrsrc.Section{
						{Index: 1},
						{Index: 2},
					},
				},
			},
		}

		_, ok, err := requireSingleSectionByType(f, "STRG")
		if err == nil {
			t.Fatalf("requireSingleSectionByType many-match error = nil, want non-nil")
		}
		if ok {
			t.Fatalf("requireSingleSectionByType many-match ok = true, want false")
		}
	})
}

func TestMutatorStrictField(t *testing.T) {
	m := Mutator{Strict: true}
	if !m.Strict {
		t.Fatalf("Mutator.Strict = false, want true")
	}
}

// TestMutationErrorErrorFormatBranches walks every formatting arm of
// MutationError.Error so the per-message variant table doesn't sit
// unexercised. The branches differ only by which of FourCC, Offset and
// Err are populated.
func TestMutationErrorErrorFormatBranches(t *testing.T) {
	underlying := errors.New("boom")
	cases := []struct {
		name     string
		err      MutationError
		contains []string
	}{
		{
			name:     "fourcc offset and underlying",
			err:      MutationError{FourCC: "STRG", Offset: 32, Cause: ErrCodecEncode, Err: underlying},
			contains: []string{"STRG", "offset 32", "boom"},
		},
		{
			name:     "fourcc and underlying",
			err:      MutationError{FourCC: "vers", Cause: ErrCodecDecode, Err: underlying},
			contains: []string{"vers", "boom"},
		},
		{
			name:     "fourcc and offset only",
			err:      MutationError{FourCC: "STRG", Offset: 16, Cause: ErrTargetMissing},
			contains: []string{"STRG", "offset 16"},
		},
		{
			name:     "fourcc only",
			err:      MutationError{FourCC: "vers", Cause: ErrTargetAmbiguous},
			contains: []string{"vers"},
		},
		{
			name:     "underlying only",
			err:      MutationError{Cause: ErrMutation, Err: underlying},
			contains: []string{"boom"},
		},
		{
			name:     "cause only",
			err:      MutationError{Cause: ErrNilFile},
			contains: []string{"nil file"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Error() = %q, want substring %q", got, want)
				}
			}
		})
	}
}

// TestMutationErrorUnwrapMatchesCauseAndUnderlying ensures errors.Is
// reaches both the sentinel Cause and the optional underlying Err.
func TestMutationErrorUnwrapMatchesCauseAndUnderlying(t *testing.T) {
	underlying := errors.New("inner")
	wrapped := &MutationError{Cause: ErrCodecEncode, Err: underlying}

	if !errors.Is(wrapped, ErrCodecEncode) {
		t.Errorf("errors.Is(MutationError, ErrCodecEncode) = false, want true")
	}
	if !errors.Is(wrapped, underlying) {
		t.Errorf("errors.Is(MutationError, underlying) = false, want true")
	}

	bare := &MutationError{Cause: ErrTargetMissing}
	if !errors.Is(bare, ErrTargetMissing) {
		t.Errorf("errors.Is(bareMutationError, ErrTargetMissing) = false, want true")
	}
}

// TestEffectiveRegistryFallsBackToDefault confirms that a zero-value
// Mutator picks up the package-level shipped registry (the codec lookup
// fallback in effectiveRegistry).
func TestEffectiveRegistryFallsBackToDefault(t *testing.T) {
	m := Mutator{}
	r := m.effectiveRegistry()
	if r == nil {
		t.Fatalf("effectiveRegistry() = nil, want default registry")
	}
	if !r.Has("STRG") {
		t.Errorf("default registry missing STRG codec")
	}

	custom := codecs.New()
	m2 := Mutator{registry: custom}
	if got := m2.effectiveRegistry(); got != custom {
		t.Errorf("custom registry not honoured: got %p want %p", got, custom)
	}
}
