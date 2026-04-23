package lvmeta

import (
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
