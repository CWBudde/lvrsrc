package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildHeapTagReportFromCorpus(t *testing.T) {
	r, err := BuildHeapTagReport()
	if err != nil {
		t.Fatalf("BuildHeapTagReport() error = %v", err)
	}
	if r.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", r.SchemaVersion)
	}
	if r.Summary.FixtureCount != 40 {
		t.Fatalf("FixtureCount = %d, want 40", r.Summary.FixtureCount)
	}
	if r.Summary.HeapTreeCount == 0 {
		t.Fatal("HeapTreeCount = 0, want non-zero")
	}
	if r.Summary.TotalNodes == 0 {
		t.Fatal("TotalNodes = 0, want non-zero")
	}
	if r.Summary.ReportedNodes == 0 {
		t.Fatal("ReportedNodes = 0, want non-zero")
	}
	if r.Summary.ReportedTagCount != len(r.Tags) {
		t.Fatalf("ReportedTagCount = %d, len(Tags) = %d", r.Summary.ReportedTagCount, len(r.Tags))
	}
	if r.Summary.StatusNodes["named-only"] == 0 {
		t.Fatal("named-only status count = 0, want non-zero")
	}
	if r.Summary.StatusNodes["class-semantic-open"] == 0 {
		t.Fatal("class-semantic-open status count = 0, want non-zero")
	}

	for i := 1; i < len(r.Tags); i++ {
		if r.Tags[i-1].Count < r.Tags[i].Count {
			t.Fatalf("tags are not sorted by descending count at index %d: %d < %d",
				i, r.Tags[i-1].Count, r.Tags[i].Count)
		}
	}

	wire := findHeapTag(t, r, "OF__compressedWireTable")
	if wire.Status != "partial" {
		t.Fatalf("OF__compressedWireTable status = %q, want partial", wire.Status)
	}
	if wire.Resources["BDHb"] == 0 {
		t.Fatalf("OF__compressedWireTable BDHb resources = 0, want non-zero")
	}
	if len(wire.Fixtures) == 0 {
		t.Fatal("OF__compressedWireTable fixture provenance is empty")
	}

	rect := findHeapTag(t, r, "OF__contRect")
	if rect.Status != "rect-role-open" {
		t.Fatalf("OF__contRect status = %q, want rect-role-open", rect.Status)
	}

	if got := findHeapTagOptional(r, "OF__bounds"); got != nil {
		t.Fatalf("OF__bounds appeared in unresolved report: %+v", *got)
	}
}

func TestRenderHeapTagReportMarkdownIncludesWorkQueue(t *testing.T) {
	r, err := BuildHeapTagReport()
	if err != nil {
		t.Fatalf("BuildHeapTagReport() error = %v", err)
	}
	md := RenderHeapTagReportMarkdown(r)
	for _, want := range []string{
		"# Heap Tag Semantic Gaps",
		"Status meanings:",
		"`OF__compressedWireTable`",
		"partial",
		"rect-role-open",
		"Parent contexts",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("RenderHeapTagReportMarkdown() missing %q", want)
		}
	}
}

func TestGeneratedHeapTagArtifactsStayInSync(t *testing.T) {
	r, err := BuildHeapTagReport()
	if err != nil {
		t.Fatalf("BuildHeapTagReport() error = %v", err)
	}
	tests := []struct {
		path string
		want string
	}{
		{
			path: filepath.Join("..", "..", "docs", "generated", "heap-tag-gaps.json"),
			want: RenderHeapTagReportJSON(r),
		},
		{
			path: filepath.Join("..", "..", "docs", "generated", "heap-tag-gaps.md"),
			want: RenderHeapTagReportMarkdown(r),
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

func findHeapTag(t *testing.T, r HeapTagReport, name string) HeapTagGap {
	t.Helper()
	if gap := findHeapTagOptional(r, name); gap != nil {
		return *gap
	}
	t.Fatalf("heap tag %q not found", name)
	return HeapTagGap{}
}

func findHeapTagOptional(r HeapTagReport, name string) *HeapTagGap {
	for i := range r.Tags {
		if r.Tags[i].Name == name {
			return &r.Tags[i]
		}
	}
	return nil
}
