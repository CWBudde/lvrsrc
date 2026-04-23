package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildManifestFromCorpus(t *testing.T) {
	m, err := BuildManifest()
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	if m.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", m.SchemaVersion)
	}
	if m.Corpus.FixtureCount != 21 {
		t.Fatalf("Corpus.FixtureCount = %d, want 21", m.Corpus.FixtureCount)
	}
	if m.Corpus.ResourceTypeCount != 27 {
		t.Fatalf("Corpus.ResourceTypeCount = %d, want 27", m.Corpus.ResourceTypeCount)
	}
	if m.Summary.TypedCodecCount != 5 {
		t.Fatalf("Summary.TypedCodecCount = %d, want 5", m.Summary.TypedCodecCount)
	}
	if m.Summary.TypedResourceTypes != 5 {
		t.Fatalf("Summary.TypedResourceTypes = %d, want 5", m.Summary.TypedResourceTypes)
	}
	if m.Summary.OpaqueResourceTypes != 22 {
		t.Fatalf("Summary.OpaqueResourceTypes = %d, want 22", m.Summary.OpaqueResourceTypes)
	}

	if len(m.Resources) != 27 {
		t.Fatalf("len(Resources) = %d, want 27", len(m.Resources))
	}
	if m.Resources[0].FourCC != "BDEx" {
		t.Fatalf("Resources[0].FourCC = %q, want %q", m.Resources[0].FourCC, "BDEx")
	}
	if m.Resources[len(m.Resources)-1].FourCC != "vers" {
		t.Fatalf("Resources[last].FourCC = %q, want %q", m.Resources[len(m.Resources)-1].FourCC, "vers")
	}

	strg := findResource(t, m, "STRG")
	if !strg.Typed.Decode || !strg.Typed.Encode || !strg.Typed.Validate {
		t.Fatalf("STRG typed support = %+v, want all true", strg.Typed)
	}
	if strg.SafetyTier != "Tier 2" {
		t.Fatalf("STRG SafetyTier = %q, want %q", strg.SafetyTier, "Tier 2")
	}
	if strg.Package != "internal/codecs/strg" {
		t.Fatalf("STRG Package = %q, want %q", strg.Package, "internal/codecs/strg")
	}
	if strg.CorpusFixtures != 4 {
		t.Fatalf("STRG CorpusFixtures = %d, want 4", strg.CorpusFixtures)
	}

	mono := findResource(t, m, "ICON")
	if !mono.Typed.Decode || !mono.Typed.Encode || !mono.Typed.Validate {
		t.Fatalf("ICON typed support = %+v, want all true", mono.Typed)
	}
	if mono.SafetyTier != "Tier 2" {
		t.Fatalf("ICON SafetyTier = %q, want %q", mono.SafetyTier, "Tier 2")
	}
	if mono.Package != "internal/codecs/icon" {
		t.Fatalf("ICON Package = %q, want %q", mono.Package, "internal/codecs/icon")
	}
	if mono.CorpusFixtures != 21 {
		t.Fatalf("ICON CorpusFixtures = %d, want 21", mono.CorpusFixtures)
	}

	vers := findResource(t, m, "vers")
	if !vers.Typed.Decode || !vers.Typed.Encode || !vers.Typed.Validate {
		t.Fatalf("vers typed support = %+v, want all true", vers.Typed)
	}
	if vers.SafetyTier != "Tier 2" {
		t.Fatalf("vers SafetyTier = %q, want %q", vers.SafetyTier, "Tier 2")
	}
	if vers.Package != "internal/codecs/vers" {
		t.Fatalf("vers Package = %q, want %q", vers.Package, "internal/codecs/vers")
	}
	if vers.CorpusFixtures != 21 {
		t.Fatalf("vers CorpusFixtures = %d, want 21", vers.CorpusFixtures)
	}

	bdpw := findResource(t, m, "BDPW")
	if bdpw.Typed.Decode || bdpw.Typed.Encode || bdpw.Typed.Validate {
		t.Fatalf("BDPW typed support = %+v, want all false", bdpw.Typed)
	}
	if bdpw.SafetyTier != "Opaque" {
		t.Fatalf("BDPW SafetyTier = %q, want %q", bdpw.SafetyTier, "Opaque")
	}
	if bdpw.Package != "internal/codecs (fallback)" {
		t.Fatalf("BDPW Package = %q, want %q", bdpw.Package, "internal/codecs (fallback)")
	}
	if bdpw.CorpusFixtures != 10 {
		t.Fatalf("BDPW CorpusFixtures = %d, want 10", bdpw.CorpusFixtures)
	}
}

func TestGeneratedArtifactsStayInSync(t *testing.T) {
	m, err := BuildManifest()
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{
			path: filepath.Join("..", "..", "docs", "generated", "resource-coverage.json"),
			want: RenderManifestJSON(m),
		},
		{
			path: filepath.Join("..", "..", "docs", "generated", "resource-coverage.md"),
			want: RenderMarkdown(m),
		},
		{
			path: filepath.Join("..", "..", "docs", "generated", "resource-coverage-badge.svg"),
			want: RenderBadgeSVG(m),
		},
	}

	for _, tc := range tests {
		t.Run(filepath.Base(tc.path), func(t *testing.T) {
			gotBytes, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", tc.path, err)
			}
			got := string(gotBytes)
			if got != tc.want {
				t.Fatalf("%s is out of date; regenerate coverage artifacts", tc.path)
			}
		})
	}
}

func TestRenderMarkdownIncludesCoverageSummary(t *testing.T) {
	m, err := BuildManifest()
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	md := RenderMarkdown(m)
	for _, want := range []string{
		"# Resource Coverage",
		"Typed coverage: 5/27 resource types",
		"`ICON`",
		"`icl4`",
		"`icl8`",
		"`STRG`",
		"`vers`",
		"`BDPW`",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("RenderMarkdown() missing %q", want)
		}
	}
}

func findResource(t *testing.T, m Manifest, fourCC string) Resource {
	t.Helper()
	for _, r := range m.Resources {
		if r.FourCC == fourCC {
			return r
		}
	}
	t.Fatalf("resource %q not found", fourCC)
	return Resource{}
}
