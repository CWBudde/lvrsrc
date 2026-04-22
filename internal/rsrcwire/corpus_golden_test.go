package rsrcwire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/golden"
)

// TestCorpusGolden asserts a structural snapshot AND a round-trip byte-diff
// count for every corpus fixture. Golden files live at
// testdata/golden/<fixture>.golden.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./internal/rsrcwire -run TestCorpusGolden
//
// The snapshot catches regressions in Parse output; the byte-diff count
// tracks Serialize fidelity against the PLAN.md Phase 2 "byte-exact
// preserving writer" goal. When diffs drop to 0, this test will catch
// any future backslide.
func TestCorpusGolden(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(corpus.Path("*.vi"))
	if err != nil {
		t.Fatalf("glob vi: %v", err)
	}
	ctlPaths, err := filepath.Glob(corpus.Path("*.ctl"))
	if err != nil {
		t.Fatalf("glob ctl: %v", err)
	}
	paths = append(paths, ctlPaths...)
	sort.Strings(paths)

	if len(paths) < 20 {
		t.Fatalf("corpus has %d fixtures; expected ≥20", len(paths))
	}

	for _, p := range paths {
		p := p
		name := filepath.Base(p)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", p, err)
			}

			parsed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", p, err)
			}

			serialized, err := Serialize(parsed)
			if err != nil {
				t.Fatalf("Serialize(%q) error = %v", p, err)
			}

			snap := buildSnapshot(parsed, data, serialized)
			got, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				t.Fatalf("marshal snapshot: %v", err)
			}
			got = append(got, '\n')

			goldenPath := filepath.Join("testdata", "golden", name+".golden.json")
			golden.Assert(t, goldenPath, got)
		})
	}
}

type corpusSnapshot struct {
	Header     headerSnapshot    `json:"header"`
	Kind       FileKind          `json:"kind"`
	BlockCount int               `json:"block_count"`
	Blocks     []blockSnapshot   `json:"blocks"`
	Names      []nameSnapshot    `json:"names"`
	RawTailLen int               `json:"raw_tail_len"`
	Issues     []issueSnapshot   `json:"issues"`
	RoundTrip  roundTripSnapshot `json:"round_trip"`
}

type headerSnapshot struct {
	Magic         string `json:"magic"`
	FormatVersion uint16 `json:"format_version"`
	Type          string `json:"type"`
	Creator       string `json:"creator"`
	InfoOffset    uint32 `json:"info_offset"`
	InfoSize      uint32 `json:"info_size"`
	DataOffset    uint32 `json:"data_offset"`
	DataSize      uint32 `json:"data_size"`
}

type blockSnapshot struct {
	Type     string            `json:"type"`
	Offset   uint32            `json:"offset"`
	Sections []sectionSnapshot `json:"sections"`
}

type sectionSnapshot struct {
	Index      int32  `json:"index"`
	NameOffset uint32 `json:"name_offset"`
	PayloadLen int    `json:"payload_len"`
	Name       string `json:"name,omitempty"`
}

type nameSnapshot struct {
	Offset uint32 `json:"offset"`
	Value  string `json:"value"`
}

type issueSnapshot struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
}

type roundTripSnapshot struct {
	SizeMatch       bool  `json:"size_match"`
	InBytes         int   `json:"in_bytes"`
	OutBytes        int   `json:"out_bytes"`
	ByteDiffs       int   `json:"byte_diffs"`
	FirstDiffOffset int64 `json:"first_diff_offset"` // -1 if no diff
}

func buildSnapshot(f *File, in, out []byte) corpusSnapshot {
	snap := corpusSnapshot{
		Header: headerSnapshot{
			Magic:         sanitizeMagic(f.Header.Magic),
			FormatVersion: f.Header.FormatVersion,
			Type:          f.Header.Type,
			Creator:       f.Header.Creator,
			InfoOffset:    f.Header.InfoOffset,
			InfoSize:      f.Header.InfoSize,
			DataOffset:    f.Header.DataOffset,
			DataSize:      f.Header.DataSize,
		},
		Kind:       f.Kind,
		BlockCount: len(f.Blocks),
		RawTailLen: len(f.RawTail),
	}

	for _, b := range f.Blocks {
		bs := blockSnapshot{Type: b.Type, Offset: b.Offset}
		for _, s := range b.Sections {
			bs.Sections = append(bs.Sections, sectionSnapshot{
				Index:      s.Index,
				NameOffset: s.NameOffset,
				PayloadLen: len(s.Payload),
				Name:       s.Name,
			})
		}
		snap.Blocks = append(snap.Blocks, bs)
	}

	for _, n := range f.Names {
		snap.Names = append(snap.Names, nameSnapshot{Offset: n.Offset, Value: n.Value})
	}

	for _, is := range f.ParseIssues {
		snap.Issues = append(snap.Issues, issueSnapshot{
			Severity: is.Severity,
			Code:     is.Code,
			Message:  is.Message,
		})
	}

	snap.RoundTrip = computeRoundTrip(in, out)
	return snap
}

func computeRoundTrip(in, out []byte) roundTripSnapshot {
	rt := roundTripSnapshot{
		SizeMatch:       len(in) == len(out),
		InBytes:         len(in),
		OutBytes:        len(out),
		FirstDiffOffset: -1,
	}

	if bytes.Equal(in, out) {
		return rt
	}

	minLen := len(in)
	if len(out) < minLen {
		minLen = len(out)
	}
	for i := 0; i < minLen; i++ {
		if in[i] != out[i] {
			if rt.FirstDiffOffset < 0 {
				rt.FirstDiffOffset = int64(i)
			}
			rt.ByteDiffs++
		}
	}
	// length tail counts as diffs too
	rt.ByteDiffs += abs(len(in) - len(out))
	return rt
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sanitizeMagic replaces non-printable bytes in the magic with \xNN so the
// JSON snapshot stays readable and diff-friendly.
func sanitizeMagic(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0x20 && r < 0x7f {
			b.WriteRune(r)
			continue
		}
		fmt.Fprintf(&b, "\\x%02x", r)
	}
	return b.String()
}
