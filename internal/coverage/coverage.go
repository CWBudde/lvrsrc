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
	"github.com/CWBudde/lvrsrc/internal/codecs/bdex"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdhb"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdpw"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdse"
	"github.com/CWBudde/lvrsrc/internal/codecs/conpane"
	"github.com/CWBudde/lvrsrc/internal/codecs/dthp"
	"github.com/CWBudde/lvrsrc/internal/codecs/fpex"
	"github.com/CWBudde/lvrsrc/internal/codecs/fphb"
	"github.com/CWBudde/lvrsrc/internal/codecs/fpse"
	"github.com/CWBudde/lvrsrc/internal/codecs/ftab"
	"github.com/CWBudde/lvrsrc/internal/codecs/hist"
	"github.com/CWBudde/lvrsrc/internal/codecs/icon"
	"github.com/CWBudde/lvrsrc/internal/codecs/libd"
	"github.com/CWBudde/lvrsrc/internal/codecs/libn"
	"github.com/CWBudde/lvrsrc/internal/codecs/lifp"
	"github.com/CWBudde/lvrsrc/internal/codecs/livi"
	"github.com/CWBudde/lvrsrc/internal/codecs/lvsr"
	"github.com/CWBudde/lvrsrc/internal/codecs/muid"
	"github.com/CWBudde/lvrsrc/internal/codecs/rtsg"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/codecs/vctp"
	"github.com/CWBudde/lvrsrc/internal/codecs/vers"
	"github.com/CWBudde/lvrsrc/internal/codecs/vits"
	"github.com/CWBudde/lvrsrc/internal/codecs/vpdp"
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
	FixtureCount      int           `json:"fixtureCount"`
	ResourceTypeCount int           `json:"resourceTypeCount"`
	Breadth           CorpusBreadth `json:"breadth"`
}

type CorpusBreadth struct {
	FileKinds            map[string]int `json:"fileKinds"`
	FileExtensions       map[string]int `json:"fileExtensions"`
	FormatVersions       map[string]int `json:"formatVersions"`
	LabVIEWVersions      map[string]int `json:"labviewVersions"`
	Platforms            map[string]int `json:"platforms"`
	TextEncodings        map[string]int `json:"textEncodings"`
	PasswordProtection   map[string]int `json:"passwordProtection"`
	LockedFlags          map[string]int `json:"lockedFlags"`
	SeparateCompiledCode map[string]int `json:"separateCompiledCode"`
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
	CorpusSections int          `json:"corpusSections"`
	CorpusBytes    int          `json:"corpusBytes"`
	Typed          TypedSupport `json:"typed"`
	SafetyTier     string       `json:"safetyTier"`
	Package        string       `json:"package"`
	ReadVersions   VersionRange `json:"readVersions"`
	WriteVersions  VersionRange `json:"writeVersions"`
	Disposition    Disposition  `json:"disposition"`
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

// Disposition records how much semantic meaning the project currently claims
// for a resource's payload bytes. It is intentionally separate from typed
// codec coverage: a resource can have a safe round-trip codec while still
// carrying documented opaque spans.
type Disposition struct {
	Status     string   `json:"status"`
	Semantic   []string `json:"semantic,omitempty"`
	Reserved   []string `json:"reserved,omitempty"`
	Compressed []string `json:"compressed,omitempty"`
	Opaque     []string `json:"opaque,omitempty"`
	Next       []string `json:"next,omitempty"`
}

type codecSpec struct {
	codec       codecs.ResourceCodec
	packagePath string
}

type resourceObservation struct {
	Fixtures int
	Sections int
	Bytes    int
}

type corpusScan struct {
	fixtures int
	observed map[string]resourceObservation
	breadth  CorpusBreadth
}

// BuildManifest derives resource coverage from the committed test corpus and
// the currently shipped typed codecs.
func BuildManifest() (Manifest, error) {
	scan, err := scanCorpus(corpusDir())
	if err != nil {
		return Manifest{}, err
	}
	observed := scan.observed

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
			CorpusFixtures: observed[fourCC].Fixtures,
			CorpusSections: observed[fourCC].Sections,
			CorpusBytes:    observed[fourCC].Bytes,
			SafetyTier:     "Opaque",
			Package:        "internal/codecs (fallback)",
			ReadVersions:   VersionRange{Min: 0, Max: 0},
			WriteVersions:  VersionRange{Min: 0, Max: 0},
			Disposition:    semanticDisposition(fourCC),
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
			FixtureCount:      scan.fixtures,
			ResourceTypeCount: totalTypes,
			Breadth:           scan.breadth,
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
	b.WriteString("## Corpus Breadth\n\n")
	writeCountSummary(&b, "File kinds", m.Corpus.Breadth.FileKinds)
	writeCountSummary(&b, "File extensions", m.Corpus.Breadth.FileExtensions)
	writeCountSummary(&b, "RSRC format versions", m.Corpus.Breadth.FormatVersions)
	writeCountSummary(&b, "LabVIEW versions", m.Corpus.Breadth.LabVIEWVersions)
	writeCountSummary(&b, "Platforms", m.Corpus.Breadth.Platforms)
	writeCountSummary(&b, "Text encodings", m.Corpus.Breadth.TextEncodings)
	writeCountSummary(&b, "Password protection", m.Corpus.Breadth.PasswordProtection)
	writeCountSummary(&b, "LVSR locked flag", m.Corpus.Breadth.LockedFlags)
	writeCountSummary(&b, "Separate compiled code", m.Corpus.Breadth.SeparateCompiledCode)
	b.WriteString("\n## Resource Table\n\n")
	headers := []string{
		"FourCC",
		"Corpus fixtures",
		"Sections",
		"Bytes",
		"Typed decode",
		"Typed encode",
		"Typed validate",
		"Byte disposition",
		"Safety",
		"Package",
		"Read versions",
		"Write versions",
	}
	alignments := []markdownTableAlignment{
		markdownAlignLeft,
		markdownAlignRight,
		markdownAlignRight,
		markdownAlignRight,
		markdownAlignCenter,
		markdownAlignCenter,
		markdownAlignCenter,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignLeft,
	}
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	rows := make([][]string, 0, len(m.Resources))
	for _, r := range m.Resources {
		row := []string{
			fmt.Sprintf("`%s`", r.FourCC),
			fmt.Sprintf("%d", r.CorpusFixtures),
			fmt.Sprintf("%d", r.CorpusSections),
			fmt.Sprintf("%d", r.CorpusBytes),
			yesNo(r.Typed.Decode),
			yesNo(r.Typed.Encode),
			yesNo(r.Typed.Validate),
			r.Disposition.Status,
			r.SafetyTier,
			fmt.Sprintf("`%s`", r.Package),
			formatVersionRange(r.ReadVersions),
			formatVersionRange(r.WriteVersions),
		}
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
		rows = append(rows, row)
	}
	writeMarkdownTable(&b, headers, alignments, widths, rows)
	b.WriteString("\n## Byte Disposition\n\n")
	b.WriteString("Status values are semantic byte-coverage claims, not codec availability. `structural` means the stable envelope is decoded but important inner bytes remain raw; `partial` means selected fields have semantic projections; `opaque-preserving` means payload bytes are intentionally retained without field meanings; `undocumented` is a failing coverage gap.\n\n")
	for _, r := range m.Resources {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", r.FourCC))
		b.WriteString(fmt.Sprintf("- Status: %s\n", r.Disposition.Status))
		writeDispositionList(&b, "Semantic", r.Disposition.Semantic)
		writeDispositionList(&b, "Reserved/padding", r.Disposition.Reserved)
		writeDispositionList(&b, "Compressed/checksum", r.Disposition.Compressed)
		writeDispositionList(&b, "Opaque", r.Disposition.Opaque)
		writeDispositionList(&b, "Next", r.Disposition.Next)
		b.WriteString("\n")
	}
	return b.String()
}

func writeCountSummary(b *strings.Builder, label string, counts map[string]int) {
	b.WriteString(fmt.Sprintf("- %s: %s\n", label, formatCounts(counts)))
}

func formatCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func writeDispositionList(b *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("- %s: %s\n", label, strings.Join(values, "; ")))
}

type markdownTableAlignment int

const (
	markdownAlignLeft markdownTableAlignment = iota
	markdownAlignRight
	markdownAlignCenter
)

func writeMarkdownTable(b *strings.Builder, headers []string, alignments []markdownTableAlignment, widths []int, rows [][]string) {
	writeMarkdownHeaderRow(b, headers, widths)
	writeMarkdownSeparatorRow(b, alignments, widths)
	for _, row := range rows {
		writeMarkdownDataRow(b, row, alignments, widths)
	}
}

func writeMarkdownHeaderRow(b *strings.Builder, row []string, widths []int) {
	b.WriteString("|")
	for i, cell := range row {
		b.WriteString(" ")
		b.WriteString(padRight(cell, widths[i]))
		b.WriteString(" |")
	}
	b.WriteString("\n")
}

func writeMarkdownSeparatorRow(b *strings.Builder, alignments []markdownTableAlignment, widths []int) {
	b.WriteString("|")
	for i, alignment := range alignments {
		b.WriteString(" ")
		b.WriteString(markdownSeparator(alignment, widths[i]))
		b.WriteString(" |")
	}
	b.WriteString("\n")
}

func writeMarkdownDataRow(b *strings.Builder, row []string, alignments []markdownTableAlignment, widths []int) {
	b.WriteString("|")
	for i, cell := range row {
		b.WriteString(" ")
		b.WriteString(padMarkdownCell(cell, widths[i], alignments[i]))
		b.WriteString(" |")
	}
	b.WriteString("\n")
}

func markdownSeparator(alignment markdownTableAlignment, width int) string {
	if width < 1 {
		width = 1
	}
	switch alignment {
	case markdownAlignRight:
		if width == 1 {
			return ":"
		}
		return strings.Repeat("-", width-1) + ":"
	case markdownAlignCenter:
		if width <= 2 {
			return ":-:"
		}
		return ":" + strings.Repeat("-", width-2) + ":"
	default:
		return strings.Repeat("-", width)
	}
}

func padMarkdownCell(value string, width int, alignment markdownTableAlignment) string {
	switch alignment {
	case markdownAlignRight:
		return padLeft(value, width)
	case markdownAlignCenter:
		return padCenter(value, width)
	default:
		return padRight(value, width)
	}
}

func padLeft(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return strings.Repeat(" ", width-len(value)) + value
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func padCenter(value string, width int) string {
	if len(value) >= width {
		return value
	}
	padding := width - len(value)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
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
	{codec: libd.Codec{}, packagePath: "internal/codecs/libd"},
	{codec: lifp.Codec{}, packagePath: "internal/codecs/lifp"},
	{codec: strg.Codec{}, packagePath: "internal/codecs/strg"},
	{codec: vers.Codec{}, packagePath: "internal/codecs/vers"},
	{codec: vctp.Codec{}, packagePath: "internal/codecs/vctp"},
	{codec: lvsr.Codec{}, packagePath: "internal/codecs/lvsr"},
	{codec: muid.Codec{}, packagePath: "internal/codecs/muid"},
	{codec: fpse.Codec{}, packagePath: "internal/codecs/fpse"},
	{codec: bdse.Codec{}, packagePath: "internal/codecs/bdse"},
	{codec: vpdp.Codec{}, packagePath: "internal/codecs/vpdp"},
	{codec: dthp.Codec{}, packagePath: "internal/codecs/dthp"},
	{codec: rtsg.Codec{}, packagePath: "internal/codecs/rtsg"},
	{codec: libn.Codec{}, packagePath: "internal/codecs/libn"},
	{codec: hist.Codec{}, packagePath: "internal/codecs/hist"},
	{codec: bdpw.Codec{}, packagePath: "internal/codecs/bdpw"},
	{codec: fpex.Codec{}, packagePath: "internal/codecs/fpex"},
	{codec: bdex.Codec{}, packagePath: "internal/codecs/bdex"},
	{codec: ftab.Codec{}, packagePath: "internal/codecs/ftab"},
	{codec: vits.Codec{}, packagePath: "internal/codecs/vits"},
	{codec: livi.Codec{}, packagePath: "internal/codecs/livi"},
	{codec: fphb.Codec{}, packagePath: "internal/codecs/fphb"},
	{codec: bdhb.Codec{}, packagePath: "internal/codecs/bdhb"},
}

func scanCorpus(dir string) (corpusScan, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return corpusScan{}, fmt.Errorf("coverage: read corpus dir: %w", err)
	}

	fixtures := 0
	observed := make(map[string]resourceObservation)
	breadth := newCorpusBreadth()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" && ext != ".llb" {
			continue
		}

		fixtures++
		f, err := lvrsrc.Open(filepath.Join(dir, name), lvrsrc.OpenOptions{})
		if err != nil {
			return corpusScan{}, fmt.Errorf("coverage: parse corpus fixture %q: %w", name, err)
		}
		recordFixtureBreadth(&breadth, f, ext)

		seen := make(map[string]struct{})
		for _, block := range f.Blocks {
			seen[block.Type] = struct{}{}
			obs := observed[block.Type]
			obs.Sections += len(block.Sections)
			for _, section := range block.Sections {
				obs.Bytes += len(section.Payload)
			}
			observed[block.Type] = obs
		}
		for fourCC := range seen {
			obs := observed[fourCC]
			obs.Fixtures++
			observed[fourCC] = obs
		}
	}

	return corpusScan{fixtures: fixtures, observed: observed, breadth: breadth}, nil
}

func newCorpusBreadth() CorpusBreadth {
	return CorpusBreadth{
		FileKinds:            map[string]int{},
		FileExtensions:       map[string]int{},
		FormatVersions:       map[string]int{},
		LabVIEWVersions:      map[string]int{},
		Platforms:            map[string]int{},
		TextEncodings:        map[string]int{},
		PasswordProtection:   map[string]int{},
		LockedFlags:          map[string]int{},
		SeparateCompiledCode: map[string]int{},
	}
}

func recordFixtureBreadth(b *CorpusBreadth, f *lvrsrc.File, ext string) {
	increment(b.FileKinds, string(f.Kind))
	increment(b.FileExtensions, ext)
	increment(b.FormatVersions, fmt.Sprintf("%d", f.Header.FormatVersion))
	increment(b.LabVIEWVersions, fixtureLabVIEWVersion(f))
	increment(b.Platforms, "unknown")
	increment(b.TextEncodings, "unknown")
	increment(b.PasswordProtection, fixturePasswordProtection(f))

	locked, separate := fixtureLVSRFlags(f)
	increment(b.LockedFlags, locked)
	increment(b.SeparateCompiledCode, separate)
}

func fixtureLabVIEWVersion(f *lvrsrc.File) string {
	for _, block := range f.Blocks {
		if block.Type != string(vers.FourCC) {
			continue
		}
		for _, section := range block.Sections {
			decoded, err := (vers.Codec{}).Decode(codecs.Context{FileVersion: f.Header.FormatVersion, Kind: f.Kind}, section.Payload)
			if err != nil {
				continue
			}
			v := decoded.(vers.Value)
			if v.Text != "" {
				return v.Text
			}
			return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
		}
	}
	return "unknown"
}

func fixturePasswordProtection(f *lvrsrc.File) string {
	for _, block := range f.Blocks {
		if block.Type != string(bdpw.FourCC) {
			continue
		}
		for _, section := range block.Sections {
			decoded, err := (bdpw.Codec{}).Decode(codecs.Context{FileVersion: f.Header.FormatVersion, Kind: f.Kind}, section.Payload)
			if err != nil {
				return "unknown"
			}
			if decoded.(bdpw.Value).HasPassword() {
				return "password"
			}
			return "empty-password"
		}
	}
	return "no-bdpw"
}

func fixtureLVSRFlags(f *lvrsrc.File) (locked, separateCompiledCode string) {
	for _, block := range f.Blocks {
		if block.Type != string(lvsr.FourCC) {
			continue
		}
		for _, section := range block.Sections {
			decoded, err := (lvsr.Codec{}).Decode(codecs.Context{FileVersion: f.Header.FormatVersion, Kind: f.Kind}, section.Payload)
			if err != nil {
				return "unknown", "unknown"
			}
			v := decoded.(lvsr.Value)
			return boolBucket(v.Locked()), boolBucket(v.SeparateCode())
		}
	}
	return "no-lvsr", "no-lvsr"
}

func boolBucket(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func increment(counts map[string]int, key string) {
	counts[key]++
}

func semanticDisposition(fourCC string) Disposition {
	if d, ok := semanticDispositions[fourCC]; ok {
		return d
	}
	return Disposition{
		Status: "undocumented",
		Next:   []string{"add a byte-disposition entry before claiming coverage"},
	}
}

var semanticDispositions = map[string]Disposition{
	"BDEx": {
		Status:   "opaque-preserving",
		Semantic: []string{"small block-diagram auxiliary envelope size and round-trip invariants"},
		Opaque:   []string{"entry meanings and correlation with BDHb remain unmapped"},
		Next:     []string{"correlate non-zero samples with block-diagram heap changes"},
	},
	"BDHb": {
		Status:     "partial",
		Semantic:   []string{"heap envelope header", "tag-stream node structure", "tag enum names", "OF__bounds", "OF__termBounds", "OF__termHotPoint", "selected compressed-wire projections", "rectangle-like heap fields", "common scalar heap fields", "common color heap fields", "structural heap container fields"},
		Compressed: []string{"zlib-compressed heap stream preserved byte-for-byte when possible"},
		Opaque:     []string{"per-class primitive fields", "structure metadata", "multi-elbow/manual/comb wire records", "container child ordering/member role semantics", "scalar bit/enum roles", "color prefix/system-color semantics", "unknown tags surfaced as Tag(N)"},
		Next:       []string{"decode per-class BD fields and finish compressed-wire topology"},
	},
	"BDPW": {
		Status:   "structural",
		Semantic: []string{"fixed-size block-diagram password/protection payload shape"},
		Opaque:   []string{"hash/salt and lockout field meanings are not mutation-safe"},
		Next:     []string{"separate protection flags from hash material with controlled password fixtures"},
	},
	"BDSE": {
		Status:   "opaque-preserving",
		Semantic: []string{"small block-diagram settings payload shape and round-trip invariants"},
		Opaque:   []string{"bit and counter meanings remain unmapped"},
		Next:     []string{"vary diagram settings one at a time"},
	},
	"CONP": {
		Status:   "partial",
		Semantic: []string{"connector-pane pointer/selector", "links to VCTP top types where present"},
		Opaque:   []string{"version-specific selector semantics beyond observed CPC2 forms"},
		Next:     []string{"expand connector-pane fixture variants and version gates"},
	},
	"CPC2": {
		Status:   "partial",
		Semantic: []string{"connector-pane count/variant value"},
		Opaque:   []string{"full pane-pattern catalog and version-specific meanings"},
		Next:     []string{"map every LabVIEW connector pattern against terminal type refs"},
	},
	"DTHP": {
		Status:   "partial",
		Semantic: []string{"default data heap index shift used to resolve heap TypeIDs into VCTP descriptors"},
		Opaque:   []string{"broader version behavior when DTHP is absent or multi-section"},
		Next:     []string{"cross-check older fixtures and every heap data-fill site"},
	},
	"FPEx": {
		Status:   "opaque-preserving",
		Semantic: []string{"small front-panel auxiliary envelope size and round-trip invariants"},
		Opaque:   []string{"entry meanings and correlation with FPHb remain unmapped"},
		Next:     []string{"correlate non-zero samples with front-panel heap changes"},
	},
	"FPHb": {
		Status:     "partial",
		Semantic:   []string{"heap envelope header", "tag-stream node structure", "tag enum names", "OF__bounds", "selected numeric data fills", "rectangle-like heap fields", "common scalar heap fields", "common color heap fields", "structural heap container fields"},
		Compressed: []string{"zlib-compressed heap stream preserved byte-for-byte when possible"},
		Opaque:     []string{"per-class visual fields", "label/caption/font/style records", "rectangle role semantics", "container child ordering/member role semantics", "scalar bit/enum roles", "color prefix/system-color semantics", "custom-control state", "unknown tags surfaced as Tag(N)"},
		Next:       []string{"decode per-class FP fields and promote additional geometry tags"},
	},
	"FPSE": {
		Status:   "opaque-preserving",
		Semantic: []string{"small front-panel settings payload shape and round-trip invariants"},
		Opaque:   []string{"bit and counter meanings remain unmapped"},
		Next:     []string{"vary panel settings one at a time"},
	},
	"FTAB": {
		Status:   "partial",
		Semantic: []string{"font table entry envelope and names"},
		Opaque:   []string{"platform-specific font attributes not fully classified"},
		Next:     []string{"add font variation fixtures across platforms"},
	},
	"HIST": {
		Status:   "structural",
		Semantic: []string{"fixed array of edit-history counters"},
		Opaque:   []string{"individual counter meanings are not confirmed"},
		Next:     []string{"diff save/edit operations to name each slot"},
	},
	"ICON": {
		Status:   "full-observed",
		Semantic: []string{"32x32 1-bit icon pixels and palette mapping"},
		Next:     []string{"keep older-version icon geometry as a version-gated check"},
	},
	"LIBN": {
		Status:   "partial",
		Semantic: []string{"library-name list envelope and Pascal-style names"},
		Opaque:   []string{"multi-library membership behavior and text encoding edge cases"},
		Next:     []string{"add multi-library and localized-name fixtures"},
	},
	"LIbd": {
		Status:   "partial",
		Semantic: []string{"link-info header", "BDHP marker", "entry count", "qualifiers", "primary/secondary PTH path refs", "typed LinkObjRef targets where ported"},
		Opaque:   []string{"Tail bytes between path refs", "unported LinkObjRef subclasses"},
		Next:     []string{"decode Tail subrecords and expand LinkObjRef target families"},
	},
	"LIfp": {
		Status:   "partial",
		Semantic: []string{"link-info header", "FPHP marker", "entry count", "qualifiers", "primary/secondary PTH path refs", "typed LinkObjRef targets where ported"},
		Opaque:   []string{"Tail bytes between path refs", "unported LinkObjRef subclasses"},
		Next:     []string{"decode Tail subrecords and expand LinkObjRef target families"},
	},
	"LIvi": {
		Status:   "partial",
		Semantic: []string{"VI link-info header", "file-kind marker", "entry count", "qualifiers", "primary/secondary PTH path refs", "typed LinkObjRef targets where ported"},
		Opaque:   []string{"Tail bytes between path refs", "unported LinkObjRef subclasses", "future file-kind markers"},
		Next:     []string{"decode Tail subrecords and broaden dependency fixture shapes"},
	},
	"LVSR": {
		Status:   "partial",
		Semantic: []string{"version word", "selected execution/debug/protection flag projections"},
		Opaque:   []string{"unsurfaced flag words and version-specific tail fields"},
		Next:     []string{"name every observed flag word and add version gates"},
	},
	"MUID": {
		Status:   "partial",
		Semantic: []string{"maximum object UID value observed at save time"},
		Opaque:   []string{"allocation scope and lifecycle semantics"},
		Next:     []string{"diff object creation/deletion sequences"},
	},
	"RTSG": {
		Status:   "structural",
		Semantic: []string{"fixed-size runtime signature payload"},
		Opaque:   []string{"signature field roles and validation algorithm"},
		Next:     []string{"vary runtime/signature-affecting settings"},
	},
	"STRG": {
		Status:   "full-observed",
		Semantic: []string{"modern LabVIEW >= 4 string-list description payload"},
		Opaque:   []string{"legacy LabVIEW < 4 count-prefixed layout is documented but untested"},
		Next:     []string{"add legacy fixtures before claiming all-version semantic coverage"},
	},
	"VCTP": {
		Status:     "partial",
		Semantic:   []string{"outer size prefix", "zlib descriptor pool", "flat descriptor headers", "flags", "FullType codes", "labels", "top-type list"},
		Compressed: []string{"compressed descriptor-pool bytes preserved; semantic diffs compare inflated pool"},
		Opaque:     []string{"type-specific Inner payloads for arrays, clusters, functions, refnums, variants, typedefs, and complex types"},
		Next:       []string{"decode each type-specific grammar and report field-level diffs"},
	},
	"VITS": {
		Status:   "partial",
		Semantic: []string{"tag entry envelope and names"},
		Opaque:   []string{"variant content bytes and per-tag meanings"},
		Next:     []string{"decode known VITS tag payloads with setting-specific fixtures"},
	},
	"VPDP": {
		Status:   "opaque-preserving",
		Semantic: []string{"observed all-zero 4-byte payload shape"},
		Opaque:   []string{"VI primitive dependency flag meanings"},
		Next:     []string{"create primitive-dependency fixtures that produce non-zero payloads"},
	},
	"icl4": {
		Status:   "full-observed",
		Semantic: []string{"32x32 4-bit icon pixels and LabVIEW palette mapping"},
		Next:     []string{"verify whether any version embeds alternate palettes"},
	},
	"icl8": {
		Status:   "full-observed",
		Semantic: []string{"32x32 8-bit icon pixels and LabVIEW palette mapping"},
		Next:     []string{"verify palette index 188 and older-version palette behavior"},
	},
	"vers": {
		Status:   "partial",
		Semantic: []string{"LabVIEW major/minor/patch/stage version stamp and text"},
		Opaque:   []string{"exact meaning of multiple version stamp roles in one file"},
		Next:     []string{"map version resource IDs to producer/save/load roles"},
	},
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
