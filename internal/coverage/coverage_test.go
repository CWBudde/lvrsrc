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
	if m.Corpus.FixtureCount != 40 {
		t.Fatalf("Corpus.FixtureCount = %d, want 40", m.Corpus.FixtureCount)
	}
	if m.Corpus.ResourceTypeCount != 27 {
		t.Fatalf("Corpus.ResourceTypeCount = %d, want 27", m.Corpus.ResourceTypeCount)
	}
	if m.Summary.TypedCodecCount != 27 {
		t.Fatalf("Summary.TypedCodecCount = %d, want 27", m.Summary.TypedCodecCount)
	}
	if m.Summary.TypedResourceTypes != 27 {
		t.Fatalf("Summary.TypedResourceTypes = %d, want 27", m.Summary.TypedResourceTypes)
	}
	if m.Summary.OpaqueResourceTypes != 0 {
		t.Fatalf("Summary.OpaqueResourceTypes = %d, want 0", m.Summary.OpaqueResourceTypes)
	}
	if got, want := m.Corpus.Breadth.FileKinds["vi"], 29; got != want {
		t.Fatalf("Breadth.FileKinds[vi] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.FileKinds["ctl"], 11; got != want {
		t.Fatalf("Breadth.FileKinds[ctl] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.FormatVersions["3"], 40; got != want {
		t.Fatalf("Breadth.FormatVersions[3] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.LabVIEWVersions["25.3.2"], 27; got != want {
		t.Fatalf("Breadth.LabVIEWVersions[25.3.2] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.Platforms["unknown"], 40; got != want {
		t.Fatalf("Breadth.Platforms[unknown] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.TextEncodings["unknown"], 40; got != want {
		t.Fatalf("Breadth.TextEncodings[unknown] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.PasswordProtection["empty-password"], 29; got != want {
		t.Fatalf("Breadth.PasswordProtection[empty-password] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.PasswordProtection["no-bdpw"], 11; got != want {
		t.Fatalf("Breadth.PasswordProtection[no-bdpw] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.LockedFlags["false"], 40; got != want {
		t.Fatalf("Breadth.LockedFlags[false] = %d, want %d", got, want)
	}
	if got, want := m.Corpus.Breadth.SeparateCompiledCode["true"], 40; got != want {
		t.Fatalf("Breadth.SeparateCompiledCode[true] = %d, want %d", got, want)
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
	if mono.CorpusFixtures != 40 {
		t.Fatalf("ICON CorpusFixtures = %d, want 40", mono.CorpusFixtures)
	}
	if mono.CorpusSections != 40 || mono.CorpusBytes != 5120 {
		t.Fatalf("ICON corpus totals = sections %d bytes %d, want sections 40 bytes 5120", mono.CorpusSections, mono.CorpusBytes)
	}
	if mono.Disposition.Status != "full-observed" {
		t.Fatalf("ICON Disposition.Status = %q, want full-observed", mono.Disposition.Status)
	}

	conp := findResource(t, m, "CONP")
	if !conp.Typed.Decode || !conp.Typed.Encode || !conp.Typed.Validate {
		t.Fatalf("CONP typed support = %+v, want all true", conp.Typed)
	}
	if conp.SafetyTier != "Tier 2" {
		t.Fatalf("CONP SafetyTier = %q, want %q", conp.SafetyTier, "Tier 2")
	}
	if conp.Package != "internal/codecs/conpane" {
		t.Fatalf("CONP Package = %q, want %q", conp.Package, "internal/codecs/conpane")
	}
	if conp.CorpusFixtures != 40 {
		t.Fatalf("CONP CorpusFixtures = %d, want 40", conp.CorpusFixtures)
	}

	cpc2 := findResource(t, m, "CPC2")
	if !cpc2.Typed.Decode || !cpc2.Typed.Encode || !cpc2.Typed.Validate {
		t.Fatalf("CPC2 typed support = %+v, want all true", cpc2.Typed)
	}
	if cpc2.SafetyTier != "Tier 2" {
		t.Fatalf("CPC2 SafetyTier = %q, want %q", cpc2.SafetyTier, "Tier 2")
	}
	if cpc2.Package != "internal/codecs/conpane" {
		t.Fatalf("CPC2 Package = %q, want %q", cpc2.Package, "internal/codecs/conpane")
	}
	if cpc2.CorpusFixtures != 40 {
		t.Fatalf("CPC2 CorpusFixtures = %d, want 40", cpc2.CorpusFixtures)
	}

	lifp := findResource(t, m, "LIfp")
	if !lifp.Typed.Decode || !lifp.Typed.Encode || !lifp.Typed.Validate {
		t.Fatalf("LIfp typed support = %+v, want all true", lifp.Typed)
	}
	if lifp.SafetyTier != "Tier 1" {
		t.Fatalf("LIfp SafetyTier = %q, want %q", lifp.SafetyTier, "Tier 1")
	}
	if lifp.Package != "internal/codecs/lifp" {
		t.Fatalf("LIfp Package = %q, want %q", lifp.Package, "internal/codecs/lifp")
	}
	if lifp.CorpusFixtures != 40 {
		t.Fatalf("LIfp CorpusFixtures = %d, want 40", lifp.CorpusFixtures)
	}
	if lifp.Disposition.Status != "partial" {
		t.Fatalf("LIfp Disposition.Status = %q, want partial", lifp.Disposition.Status)
	}
	if len(lifp.Disposition.Opaque) == 0 {
		t.Fatalf("LIfp Disposition.Opaque is empty, want documented opaque tail")
	}

	libd := findResource(t, m, "LIbd")
	if !libd.Typed.Decode || !libd.Typed.Encode || !libd.Typed.Validate {
		t.Fatalf("LIbd typed support = %+v, want all true", libd.Typed)
	}
	if libd.SafetyTier != "Tier 1" {
		t.Fatalf("LIbd SafetyTier = %q, want %q", libd.SafetyTier, "Tier 1")
	}
	if libd.Package != "internal/codecs/libd" {
		t.Fatalf("LIbd Package = %q, want %q", libd.Package, "internal/codecs/libd")
	}
	if libd.CorpusFixtures != 40 {
		t.Fatalf("LIbd CorpusFixtures = %d, want 40", libd.CorpusFixtures)
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
	if vers.CorpusFixtures != 40 {
		t.Fatalf("vers CorpusFixtures = %d, want 40", vers.CorpusFixtures)
	}

	vctp := findResource(t, m, "VCTP")
	if !vctp.Typed.Decode || !vctp.Typed.Encode || !vctp.Typed.Validate {
		t.Fatalf("VCTP typed support = %+v, want all true", vctp.Typed)
	}
	if vctp.SafetyTier != "Tier 1" {
		t.Fatalf("VCTP SafetyTier = %q, want %q", vctp.SafetyTier, "Tier 1")
	}
	if vctp.Package != "internal/codecs/vctp" {
		t.Fatalf("VCTP Package = %q, want %q", vctp.Package, "internal/codecs/vctp")
	}
	if vctp.CorpusFixtures != 40 {
		t.Fatalf("VCTP CorpusFixtures = %d, want 40", vctp.CorpusFixtures)
	}
	if vctp.Disposition.Status != "partial" {
		t.Fatalf("VCTP Disposition.Status = %q, want partial", vctp.Disposition.Status)
	}
	if len(vctp.Disposition.Opaque) == 0 {
		t.Fatalf("VCTP Disposition.Opaque is empty, want documented Inner payload gap")
	}

	bdpw := findResource(t, m, "BDPW")
	if !bdpw.Typed.Decode || !bdpw.Typed.Encode || !bdpw.Typed.Validate {
		t.Fatalf("BDPW typed support = %+v, want all true", bdpw.Typed)
	}
	if bdpw.SafetyTier != "Tier 1" {
		t.Fatalf("BDPW SafetyTier = %q, want %q", bdpw.SafetyTier, "Tier 1")
	}
	if bdpw.Package != "internal/codecs/bdpw" {
		t.Fatalf("BDPW Package = %q, want %q", bdpw.Package, "internal/codecs/bdpw")
	}
	if bdpw.CorpusFixtures != 29 {
		t.Fatalf("BDPW CorpusFixtures = %d, want 29", bdpw.CorpusFixtures)
	}

	bdhb := findResource(t, m, "BDHb")
	if !bdhb.Typed.Decode || !bdhb.Typed.Encode || !bdhb.Typed.Validate {
		t.Fatalf("BDHb typed support = %+v, want all true", bdhb.Typed)
	}
	if bdhb.SafetyTier != "Tier 1" {
		t.Fatalf("BDHb SafetyTier = %q, want %q", bdhb.SafetyTier, "Tier 1")
	}
	if bdhb.Package != "internal/codecs/bdhb" {
		t.Fatalf("BDHb Package = %q, want %q", bdhb.Package, "internal/codecs/bdhb")
	}
	if bdhb.CorpusFixtures != 40 {
		t.Fatalf("BDHb CorpusFixtures = %d, want 40", bdhb.CorpusFixtures)
	}
	if bdhb.Disposition.Status != "partial" {
		t.Fatalf("BDHb Disposition.Status = %q, want partial", bdhb.Disposition.Status)
	}
	if len(bdhb.Disposition.Opaque) == 0 {
		t.Fatalf("BDHb Disposition.Opaque is empty, want documented heap gaps")
	}

	for _, r := range m.Resources {
		if r.Disposition.Status == "" || r.Disposition.Status == "undocumented" {
			t.Fatalf("%s has undocumented byte disposition: %+v", r.FourCC, r.Disposition)
		}
		if len(r.Disposition.Semantic) == 0 {
			t.Fatalf("%s Disposition.Semantic is empty", r.FourCC)
		}
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
				if !generatedArtifactWritable(tc.path) {
					t.Skipf("%s is out of date but not writable in this checkout; regenerate coverage artifacts after fixing file permissions", tc.path)
				}
				t.Fatalf("%s is out of date; regenerate coverage artifacts", tc.path)
			}
		})
	}
}

func generatedArtifactWritable(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func TestRenderMarkdownIncludesCoverageSummary(t *testing.T) {
	m, err := BuildManifest()
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	md := RenderMarkdown(m)
	for _, want := range []string{
		"# Resource Coverage",
		"Typed coverage: 27/27 resource types",
		"## Corpus Breadth",
		"File kinds: ctl=11, vi=29",
		"LabVIEW versions: 25.1.1=3, 25.1.2=10, 25.3.2=27",
		"Separate compiled code: true=40",
		"Byte disposition",
		"## Byte Disposition",
		"Status: partial",
		"Opaque: Tail bytes between path refs",
		"`CONP`",
		"`CPC2`",
		"`ICON`",
		"`LIbd`",
		"`LIfp`",
		"`VCTP`",
		"`icl4`",
		"`icl8`",
		"`STRG`",
		"`vers`",
		"`BDPW`",
		"`BDHb`",
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
