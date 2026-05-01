package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// HeapTagReport is a corpus-derived work queue for heap semantic gaps.
type HeapTagReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	Summary       HeapTagSummary   `json:"summary"`
	Tags          []HeapTagGap     `json:"tags"`
	Generated     HeapTagGenerated `json:"generated"`
}

type HeapTagSummary struct {
	FixtureCount     int            `json:"fixtureCount"`
	HeapTreeCount    int            `json:"heapTreeCount"`
	TotalNodes       int            `json:"totalNodes"`
	ReportedNodes    int            `json:"reportedNodes"`
	ReportedTagCount int            `json:"reportedTagCount"`
	StatusNodes      map[string]int `json:"statusNodes"`
}

type HeapTagGenerated struct {
	Manifest string `json:"manifest"`
	Report   string `json:"report"`
}

type HeapTagGap struct {
	Tag            int32              `json:"tag"`
	Name           string             `json:"name"`
	Family         string             `json:"family"`
	Status         string             `json:"status"`
	Count          int                `json:"count"`
	LeafCount      int                `json:"leafCount,omitempty"`
	OpenCount      int                `json:"openCount,omitempty"`
	CloseCount     int                `json:"closeCount,omitempty"`
	ContentBytes   int                `json:"contentBytes,omitempty"`
	WireBytes      int                `json:"wireBytes,omitempty"`
	Resources      map[string]int     `json:"resources"`
	Fixtures       []HeapFixtureCount `json:"fixtures"`
	ParentContexts []string           `json:"parentContexts,omitempty"`
	Next           string             `json:"next"`
}

type HeapFixtureCount struct {
	Fixture string `json:"fixture"`
	Count   int    `json:"count"`
}

type heapTagAccumulator struct {
	gap            HeapTagGap
	fixtures       map[string]int
	parentContexts map[string]int
}

// BuildHeapTagReport scans every FPHb/BDHb tree in the corpus and reports
// heap tags whose semantics are not yet fully decoded.
func BuildHeapTagReport() (HeapTagReport, error) {
	dir := corpusDir()
	entries, err := sortedCorpusEntries(dir)
	if err != nil {
		return HeapTagReport{}, err
	}

	fixtures := 0
	treeCount := 0
	totalNodes := 0
	reportedNodes := 0
	statusNodes := map[string]int{}
	accs := map[int32]*heapTagAccumulator{}

	for _, name := range entries {
		ext := filepath.Ext(name)
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" && ext != ".llb" {
			continue
		}
		fixtures++
		f, err := lvrsrc.Open(filepath.Join(dir, name), lvrsrc.OpenOptions{})
		if err != nil {
			return HeapTagReport{}, fmt.Errorf("coverage: parse heap fixture %q: %w", name, err)
		}
		model, _ := lvvi.DecodeKnownResources(f)
		if model == nil {
			continue
		}
		for _, treeResource := range []struct {
			name string
			get  func() (lvvi.HeapTree, bool)
		}{
			{name: "FPHb", get: model.FrontPanel},
			{name: "BDHb", get: model.BlockDiagram},
		} {
			tree, ok := treeResource.get()
			if !ok {
				continue
			}
			treeCount++
			for i, n := range tree.Nodes {
				totalNodes++
				family, tagName := heapTagFamilyAndName(n)
				status, next, report := heapTagStatus(n.Tag, family)
				statusNodes[status]++
				if !report {
					continue
				}
				reportedNodes++
				acc := accs[n.Tag]
				if acc == nil {
					acc = &heapTagAccumulator{
						gap: HeapTagGap{
							Tag:       n.Tag,
							Name:      tagName,
							Family:    family,
							Status:    status,
							Resources: map[string]int{},
							Next:      next,
						},
						fixtures:       map[string]int{},
						parentContexts: map[string]int{},
					}
					accs[n.Tag] = acc
				}
				acc.gap.Count++
				acc.gap.WireBytes += n.ByteSize
				acc.gap.Resources[treeResource.name]++
				acc.fixtures[name]++
				switch n.Scope {
				case "leaf":
					acc.gap.LeafCount++
					acc.gap.ContentBytes += len(n.Content)
				case "open":
					acc.gap.OpenCount++
				case "close":
					acc.gap.CloseCount++
				}
				acc.parentContexts[heapParentContext(tree, i)]++
			}
		}
	}

	gaps := make([]HeapTagGap, 0, len(accs))
	for _, acc := range accs {
		acc.gap.Fixtures = sortedFixtureCounts(acc.fixtures)
		acc.gap.ParentContexts = topCountLabels(acc.parentContexts, 8)
		gaps = append(gaps, acc.gap)
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Count != gaps[j].Count {
			return gaps[i].Count > gaps[j].Count
		}
		if gaps[i].Status != gaps[j].Status {
			return gaps[i].Status < gaps[j].Status
		}
		if gaps[i].Name != gaps[j].Name {
			return gaps[i].Name < gaps[j].Name
		}
		return gaps[i].Tag < gaps[j].Tag
	})

	return HeapTagReport{
		SchemaVersion: 1,
		Summary: HeapTagSummary{
			FixtureCount:     fixtures,
			HeapTreeCount:    treeCount,
			TotalNodes:       totalNodes,
			ReportedNodes:    reportedNodes,
			ReportedTagCount: len(gaps),
			StatusNodes:      statusNodes,
		},
		Tags: gaps,
		Generated: HeapTagGenerated{
			Manifest: "docs/generated/heap-tag-gaps.json",
			Report:   "docs/generated/heap-tag-gaps.md",
		},
	}, nil
}

func RenderHeapTagReportJSON(r HeapTagReport) string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("coverage: marshal heap tag report: %v", err))
	}
	return string(data) + "\n"
}

func RenderHeapTagReportMarkdown(r HeapTagReport) string {
	var b strings.Builder
	b.WriteString("# Heap Tag Semantic Gaps\n\n")
	b.WriteString(fmt.Sprintf(
		"Reported %d unresolved or partial heap tags (%d nodes) across %d heap trees in %d corpus fixtures. Total heap nodes scanned: %d.\n\n",
		r.Summary.ReportedTagCount,
		r.Summary.ReportedNodes,
		r.Summary.HeapTreeCount,
		r.Summary.FixtureCount,
		r.Summary.TotalNodes,
	))
	b.WriteString("Status counts are node counts: ")
	b.WriteString(formatCounts(r.Summary.StatusNodes))
	b.WriteString(".\n\n")
	b.WriteString("Status meanings: `named-only` has an enum name but no typed payload decoder; `class-semantic-open` has a class enum but no per-class semantic decoder; `rect-role-open` has the shared rectangle payload decoded but no confirmed scene/layout role; `partial` has a typed accessor with known remaining byte roles; `unknown-tag` has no generated enum name.\n\n")

	headers := []string{"Rank", "Tag", "Name", "Family", "Status", "Nodes", "Leaf bytes", "Resources", "Fixtures", "Parent contexts", "Next"}
	alignments := []markdownTableAlignment{
		markdownAlignRight,
		markdownAlignRight,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignRight,
		markdownAlignRight,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignLeft,
		markdownAlignLeft,
	}
	rows := make([][]string, 0, len(r.Tags))
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for i, gap := range r.Tags {
		row := []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%d", gap.Tag),
			fmt.Sprintf("`%s`", gap.Name),
			gap.Family,
			gap.Status,
			fmt.Sprintf("%d", gap.Count),
			fmt.Sprintf("%d", gap.ContentBytes),
			formatCounts(gap.Resources),
			formatFixtureSample(gap.Fixtures, 4),
			strings.Join(gap.ParentContexts, ", "),
			gap.Next,
		}
		for j, cell := range row {
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
		rows = append(rows, row)
	}
	writeMarkdownTable(&b, headers, alignments, widths, rows)
	return b.String()
}

func HeapTagArtifactContents(r HeapTagReport) map[string]string {
	root := repoRoot()
	return map[string]string{
		filepath.Join(root, "docs", "generated", "heap-tag-gaps.json"): RenderHeapTagReportJSON(r),
		filepath.Join(root, "docs", "generated", "heap-tag-gaps.md"):   RenderHeapTagReportMarkdown(r),
	}
}

func sortedCorpusEntries(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("coverage: read corpus dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func heapTagFamilyAndName(n lvvi.HeapNode) (family, name string) {
	if n.Tag < 0 {
		name := heap.SystemTag(n.Tag).String()
		if !strings.Contains(name, "(") {
			return "system", name
		}
		return "unknown", fmt.Sprintf("Tag(%d)", n.Tag)
	}
	if n.Scope == "leaf" {
		if name := heap.FieldTag(n.Tag).String(); !strings.Contains(name, "(") {
			return "field", name
		}
		if name := heap.ClassTag(n.Tag).String(); !strings.Contains(name, "(") {
			return "class", name
		}
	} else {
		if name := heap.ClassTag(n.Tag).String(); !strings.Contains(name, "(") {
			return "class", name
		}
		if name := heap.FieldTag(n.Tag).String(); !strings.Contains(name, "(") {
			return "field", name
		}
	}
	return "unknown", fmt.Sprintf("Tag(%d)", n.Tag)
}

func heapTagStatus(tag int32, family string) (status, next string, report bool) {
	if family == "unknown" {
		return "unknown-tag", "identify the tag family and add it to the generated heap enum tables", true
	}
	if family == "system" {
		return "resolved", "", false
	}
	if family == "class" {
		return "class-semantic-open", "map required/optional child fields and add a per-class semantic decoder or explicit fallback", true
	}

	switch tag {
	case int32(heap.FieldTagBounds),
		int32(heap.FieldTagTermBounds),
		int32(heap.FieldTagTermHotPoint):
		return "resolved", "", false
	case int32(heap.FieldTagCompressedWireTable):
		return "partial", "finish multi-elbow, manual-chain, comb, and 4+ branch wire semantics", true
	case int32(heap.FieldTagStdNumMin),
		int32(heap.FieldTagStdNumMax),
		int32(heap.FieldTagStdNumInc):
		return "partial", "extend HeapNodeTDDataFill beyond currently resolved primitive numeric forms", true
	case int32(heap.FieldTagTypeDesc):
		return "partial", "connect every heap-local type reference to VCTP/DTHP and validate version gates", true
	}
	if lvvi.IsHeapRectTag(tag) {
		return "rect-role-open", "confirm the rectangle role with controlled fixtures before scene/layout promotion", true
	}
	return "named-only", "decode the field payload or document it as reserved/padding with fixture evidence", true
}

func heapParentContext(tree lvvi.HeapTree, nodeIdx int) string {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return "unknown"
	}
	parentIdx := tree.Nodes[nodeIdx].Parent
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) {
		return "root"
	}
	_, name := heapTagFamilyAndName(tree.Nodes[parentIdx])
	return name
}

func sortedFixtureCounts(counts map[string]int) []HeapFixtureCount {
	out := make([]HeapFixtureCount, 0, len(counts))
	for fixture, count := range counts {
		out = append(out, HeapFixtureCount{Fixture: fixture, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Fixture < out[j].Fixture
	})
	return out
}

func topCountLabels(counts map[string]int, limit int) []string {
	fixtures := sortedFixtureCounts(counts)
	if limit > 0 && len(fixtures) > limit {
		fixtures = fixtures[:limit]
	}
	out := make([]string, 0, len(fixtures))
	for _, item := range fixtures {
		out = append(out, fmt.Sprintf("%s=%d", item.Fixture, item.Count))
	}
	return out
}

func formatFixtureSample(fixtures []HeapFixtureCount, limit int) string {
	if len(fixtures) == 0 {
		return "none"
	}
	if limit <= 0 || limit > len(fixtures) {
		limit = len(fixtures)
	}
	parts := make([]string, 0, limit+1)
	for _, item := range fixtures[:limit] {
		parts = append(parts, fmt.Sprintf("%s=%d", item.Fixture, item.Count))
	}
	if len(fixtures) > limit {
		parts = append(parts, fmt.Sprintf("+%d more", len(fixtures)-limit))
	}
	return strings.Join(parts, ", ")
}
