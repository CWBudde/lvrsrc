package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/conpane"
	"github.com/CWBudde/lvrsrc/internal/codecs/icon"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/codecs/vers"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// Manifest is the machine-readable resource coverage dashboard artifact.
type Manifest struct {
	SchemaVersion int                `json:"schemaVersion"`
	Corpus        CorpusSummary      `json:"corpus"`
	Summary       Summary            `json:"summary"`
	Resources     []Resource         `json:"resources"`
	Generated     GeneratedArtifacts `json:"generated"`
}

type CorpusSummary struct {
	FixtureCount      int `json:"fixtureCount"`
	ResourceTypeCount int `json:"resourceTypeCount"`
}

type Summary struct {
	TypedCodecCount     int     `json:"typedCodecCount"`
	TypedResourceTypes  int     `json:"typedResourceTypes"`
	OpaqueResourceTypes int     `json:"opaqueResourceTypes"`
	TypedCoveragePct    float64 `json:"typedCoveragePct"`
}

type Resource struct {
	FourCC         string       `json:"fourCC"`
	CorpusFixtures int          `json:"corpusFixtures"`
	Typed          TypedSupport `json:"typed"`
	SafetyTier     string       `json:"safetyTier"`
	Package        string       `json:"package"`
	ReadVersions   VersionRange `json:"readVersions"`
	WriteVersions  VersionRange `json:"writeVersions"`
}

type TypedSupport struct {
	Decode   bool `json:"decode"`
	Encode   bool `json:"encode"`
	Validate bool `json:"validate"`
}

type VersionRange struct {
	Min uint16 `json:"min"`
	Max uint16 `json:"max"`
}

type GeneratedArtifacts struct {
	Manifest string `json:"manifest"`
	Report   string `json:"report"`
	Badge    string `json:"badge"`
}

type codecSpec struct {
	codec       codecs.ResourceCodec
	packagePath string
}

// BuildManifest derives resource coverage from the committed test corpus and
// the currently shipped typed codecs.
func BuildManifest() (Manifest, error) {
	fixtures, observed, err := scanCorpus(corpusDir())
	if err != nil {
		return Manifest{}, err
	}

	typedByFourCC := make(map[string]codecSpec, len(shippedCodecs))
	for _, spec := range shippedCodecs {
		cap := spec.codec.Capability()
		typedByFourCC[cap.FourCC] = spec
	}

	fourCCs := make([]string, 0, len(observed))
	for fourCC := range observed {
		fourCCs = append(fourCCs, fourCC)
	}
	sort.Strings(fourCCs)

	resources := make([]Resource, 0, len(fourCCs))
	typedResourceTypes := 0
	for _, fourCC := range fourCCs {
		res := Resource{
			FourCC:         fourCC,
			CorpusFixtures: observed[fourCC],
			SafetyTier:     "Opaque",
			Package:        "internal/codecs (fallback)",
			ReadVersions:   VersionRange{Min: 0, Max: 0},
			WriteVersions:  VersionRange{Min: 0, Max: 0},
		}
		if spec, ok := typedByFourCC[fourCC]; ok {
			cap := spec.codec.Capability()
			res.Typed = TypedSupport{Decode: true, Encode: true, Validate: true}
			res.SafetyTier = formatSafetyTier(cap.Safety)
			res.Package = spec.packagePath
			res.ReadVersions = VersionRange{Min: cap.ReadVersions.Min, Max: cap.ReadVersions.Max}
			res.WriteVersions = VersionRange{Min: cap.WriteVersions.Min, Max: cap.WriteVersions.Max}
			typedResourceTypes++
		}
		resources = append(resources, res)
	}

	totalTypes := len(resources)
	typedCoveragePct := 0.0
	if totalTypes > 0 {
		typedCoveragePct = float64(typedResourceTypes) * 100 / float64(totalTypes)
	}

	return Manifest{
		SchemaVersion: 1,
		Corpus: CorpusSummary{
			FixtureCount:      fixtures,
			ResourceTypeCount: totalTypes,
		},
		Summary: Summary{
			TypedCodecCount:     len(shippedCodecs),
			TypedResourceTypes:  typedResourceTypes,
			OpaqueResourceTypes: totalTypes - typedResourceTypes,
			TypedCoveragePct:    typedCoveragePct,
		},
		Resources: resources,
		Generated: GeneratedArtifacts{
			Manifest: "docs/generated/resource-coverage.json",
			Report:   "docs/generated/resource-coverage.md",
			Badge:    "docs/generated/resource-coverage-badge.svg",
		},
	}, nil
}

// RenderManifestJSON renders the machine-readable manifest.
func RenderManifestJSON(m Manifest) string {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("coverage: marshal manifest: %v", err))
	}
	return string(data) + "\n"
}

// RenderMarkdown renders a human-readable coverage report.
func RenderMarkdown(m Manifest) string {
	var b strings.Builder
	b.WriteString("# Resource Coverage\n\n")
	b.WriteString(fmt.Sprintf(
		"Typed coverage: %d/%d resource types (%.1f%%) across %d corpus fixtures.\n\n",
		m.Summary.TypedResourceTypes,
		m.Corpus.ResourceTypeCount,
		m.Summary.TypedCoveragePct,
		m.Corpus.FixtureCount,
	))
	b.WriteString("| FourCC | Corpus fixtures | Typed decode | Typed encode | Typed validate | Safety | Package | Read versions | Write versions |\n")
	b.WriteString("| ------ | --------------: | :----------: | :----------: | :------------: | ------ | ------- | ------------- | -------------- |\n")
	for _, r := range m.Resources {
		b.WriteString(fmt.Sprintf(
			"| `%s` | %d | %s | %s | %s | %s | `%s` | %s | %s |\n",
			r.FourCC,
			r.CorpusFixtures,
			yesNo(r.Typed.Decode),
			yesNo(r.Typed.Encode),
			yesNo(r.Typed.Validate),
			r.SafetyTier,
			r.Package,
			formatVersionRange(r.ReadVersions),
			formatVersionRange(r.WriteVersions),
		))
	}
	return b.String()
}

// RenderBadgeSVG renders a local badge suitable for README.md.
func RenderBadgeSVG(m Manifest) string {
	value := fmt.Sprintf("%d/%d typed", m.Summary.TypedResourceTypes, m.Corpus.ResourceTypeCount)
	left := "resource coverage"
	leftWidth := 118
	rightWidth := 62
	totalWidth := leftWidth + rightWidth
	color := badgeColor(m.Summary.TypedCoveragePct)

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
<linearGradient id="s" x2="0" y2="100%%">
<stop offset="0" stop-color="#fff" stop-opacity=".7"/>
<stop offset=".1" stop-color="#aaa" stop-opacity=".1"/>
<stop offset=".9" stop-color="#000" stop-opacity=".3"/>
<stop offset="1" stop-color="#000" stop-opacity=".5"/>
</linearGradient>
<clipPath id="r">
<rect width="%d" height="20" rx="3" fill="#fff"/>
</clipPath>
<g clip-path="url(#r)">
<rect width="%d" height="20" fill="#555"/>
<rect x="%d" width="%d" height="20" fill="%s"/>
<rect width="%d" height="20" fill="url(#s)"/>
</g>
<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
<text x="%d" y="14">%s</text>
<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
<text x="%d" y="14">%s</text>
</g>
</svg>
`, totalWidth, left, value,
		totalWidth,
		leftWidth,
		leftWidth, rightWidth, color,
		totalWidth,
		leftWidth/2, left,
		leftWidth/2, left,
		leftWidth+rightWidth/2, value,
		leftWidth+rightWidth/2, value,
	)
}

// ArtifactContents returns the generated coverage artifacts keyed by absolute
// path so tooling can write or verify them.
func ArtifactContents(m Manifest) map[string]string {
	root := repoRoot()
	return map[string]string{
		filepath.Join(root, "docs", "generated", "resource-coverage.json"):      RenderManifestJSON(m),
		filepath.Join(root, "docs", "generated", "resource-coverage.md"):        RenderMarkdown(m),
		filepath.Join(root, "docs", "generated", "resource-coverage-badge.svg"): RenderBadgeSVG(m),
	}
}

var shippedCodecs = []codecSpec{
	{codec: conpane.PointerCodec{}, packagePath: "internal/codecs/conpane"},
	{codec: conpane.CountCodec{}, packagePath: "internal/codecs/conpane"},
	{codec: icon.MonoCodec{}, packagePath: "internal/codecs/icon"},
	{codec: icon.Color4Codec{}, packagePath: "internal/codecs/icon"},
	{codec: icon.Color8Codec{}, packagePath: "internal/codecs/icon"},
	{codec: strg.Codec{}, packagePath: "internal/codecs/strg"},
	{codec: vers.Codec{}, packagePath: "internal/codecs/vers"},
}

func scanCorpus(dir string) (int, map[string]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, nil, fmt.Errorf("coverage: read corpus dir: %w", err)
	}

	fixtures := 0
	observed := make(map[string]int)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".vi" && ext != ".ctl" {
			continue
		}

		fixtures++
		f, err := lvrsrc.Open(filepath.Join(dir, name), lvrsrc.OpenOptions{})
		if err != nil {
			return 0, nil, fmt.Errorf("coverage: parse corpus fixture %q: %w", name, err)
		}

		seen := make(map[string]struct{})
		for _, block := range f.Blocks {
			seen[block.Type] = struct{}{}
		}
		for fourCC := range seen {
			observed[fourCC]++
		}
	}

	return fixtures, observed, nil
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func corpusDir() string {
	return filepath.Join(repoRoot(), "testdata", "corpus")
}

func formatSafetyTier(tier codecs.SafetyTier) string {
	switch tier {
	case codecs.SafetyTier1:
		return "Tier 1"
	case codecs.SafetyTier2:
		return "Tier 2"
	case codecs.SafetyTier3:
		return "Tier 3"
	default:
		return fmt.Sprintf("Tier %d", int(tier))
	}
}

func formatVersionRange(r VersionRange) string {
	if r.Min == 0 && r.Max == 0 {
		return "all"
	}
	if r.Max == 0 {
		return fmt.Sprintf("%d+", r.Min)
	}
	if r.Min == r.Max {
		return fmt.Sprintf("%d", r.Min)
	}
	return fmt.Sprintf("%d-%d", r.Min, r.Max)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func badgeColor(pct float64) string {
	switch {
	case pct >= 75:
		return "#4c1"
	case pct >= 50:
		return "#97ca00"
	case pct >= 25:
		return "#dfb317"
	default:
		return "#e05d44"
	}
}
