// Package lvdiff computes structural diffs between two parsed LabVIEW RSRC
// files.
//
// The diff operates on pkg/lvrsrc.File values and is intended both for
// human-readable reporting (via the CLI) and machine-readable consumption
// (JSON). It reports differences at three granularities:
//
//  1. Header-level scalar fields (primary + secondary header, block info list).
//  2. Block-level (resource-type) additions and removals, matched by FourCC.
//  3. Section-level payload differences within common blocks: size changes,
//     content hash changes, additions and removals.
//
// A fourth layer (KindDecoded) compares decoded resource values using the
// shipped typed codecs by default. Callers can override or disable that layer
// via Options.DecodedDiffers.
package lvdiff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// Kind classifies what part of the file a DiffItem describes.
type Kind string

const (
	KindHeader  Kind = "header"
	KindBlock   Kind = "block"
	KindSection Kind = "section"
	KindDecoded Kind = "decoded"
)

// Category classifies how a DiffItem changed between the two files.
type Category string

const (
	CategoryAdded    Category = "added"
	CategoryRemoved  Category = "removed"
	CategoryModified Category = "modified"
)

// DiffItem describes a single structural difference between two files.
//
// Old holds the value from the "a" side and New the value from the "b" side.
// For CategoryAdded items Old is nil; for CategoryRemoved items New is nil.
// Values are kept as typed Go values (uint16, uint32, int, string, ...) so
// that callers can emit them as JSON without format loss.
type DiffItem struct {
	Kind     Kind
	Category Category
	Path     string
	Old      any
	New      any
	Message  string
}

// Diff is the full result of comparing two files.
type Diff struct {
	Items []DiffItem
}

// IsEmpty reports whether the diff contains no items.
func (d *Diff) IsEmpty() bool {
	return d == nil || len(d.Items) == 0
}

// Filter returns the subset of items whose Kind matches kind.
func (d *Diff) Filter(kind Kind) []DiffItem {
	if d == nil {
		return nil
	}
	var out []DiffItem
	for _, it := range d.Items {
		if it.Kind == kind {
			out = append(out, it)
		}
	}
	return out
}

// ByCategory returns the subset of items whose Category matches c.
func (d *Diff) ByCategory(c Category) []DiffItem {
	if d == nil {
		return nil
	}
	var out []DiffItem
	for _, it := range d.Items {
		if it.Category == c {
			out = append(out, it)
		}
	}
	return out
}

// DecodedDiffer diffs the decoded contents of a single section pair. It is
// invoked once per matching (blockType, sectionIndex) pair whose raw payloads
// differ. Implementations return zero or more KindDecoded items.
type DecodedDiffer func(blockType string, sectionIndex int32, oldPayload, newPayload []byte) []DiffItem

// Options tunes diff behavior. The zero value produces a structural diff
// without decoded-resource comparisons.
type Options struct {
	// DecodedDiffers maps block FourCC to a function that produces
	// decoded-level diffs for that resource type. Block types not present
	// in the map are skipped (raw-payload diffing still runs regardless).
	DecodedDiffers map[string]DecodedDiffer
}

// Files returns the structural diff between a and b using the built-in
// decoded-resource differs for shipped typed codecs.
func Files(a, b *lvrsrc.File) *Diff {
	return FilesWithOptions(a, b, Options{})
}

// FilesWithOptions returns the structural diff between a and b using the
// supplied options.
func FilesWithOptions(a, b *lvrsrc.File, opts Options) *Diff {
	if opts.DecodedDiffers == nil {
		opts.DecodedDiffers = defaultDecodedDiffers()
	}

	d := &Diff{}

	switch {
	case a == nil && b == nil:
		return d
	case a == nil:
		d.Items = append(d.Items, DiffItem{
			Kind:     KindBlock,
			Category: CategoryAdded,
			Path:     "file",
			Message:  "file added (nil on left)",
		})
		return d
	case b == nil:
		d.Items = append(d.Items, DiffItem{
			Kind:     KindBlock,
			Category: CategoryRemoved,
			Path:     "file",
			Message:  "file removed (nil on right)",
		})
		return d
	}

	diffHeaders(a, b, d)
	diffBlocks(a, b, opts, d)

	return d
}

func diffHeaders(a, b *lvrsrc.File, d *Diff) {
	diffHeader("header", a.Header, b.Header, d)
	diffHeader("secondaryHeader", a.SecondaryHeader, b.SecondaryHeader, d)
	diffBlockInfoList("blockInfoList", a.BlockInfoList, b.BlockInfoList, d)
}

func diffHeader(prefix string, a, b lvrsrc.Header, d *Diff) {
	appendScalar(d, prefix+".Magic", a.Magic, b.Magic)
	appendScalar(d, prefix+".FormatVersion", a.FormatVersion, b.FormatVersion)
	appendScalar(d, prefix+".Type", a.Type, b.Type)
	appendScalar(d, prefix+".Creator", a.Creator, b.Creator)
	appendScalar(d, prefix+".InfoOffset", a.InfoOffset, b.InfoOffset)
	appendScalar(d, prefix+".InfoSize", a.InfoSize, b.InfoSize)
	appendScalar(d, prefix+".DataOffset", a.DataOffset, b.DataOffset)
	appendScalar(d, prefix+".DataSize", a.DataSize, b.DataSize)
}

func diffBlockInfoList(prefix string, a, b lvrsrc.BlockInfoList, d *Diff) {
	appendScalar(d, prefix+".DatasetInt1", a.DatasetInt1, b.DatasetInt1)
	appendScalar(d, prefix+".DatasetInt2", a.DatasetInt2, b.DatasetInt2)
	appendScalar(d, prefix+".DatasetInt3", a.DatasetInt3, b.DatasetInt3)
	appendScalar(d, prefix+".BlockInfoOffset", a.BlockInfoOffset, b.BlockInfoOffset)
	appendScalar(d, prefix+".BlockInfoSize", a.BlockInfoSize, b.BlockInfoSize)
}

func appendScalar(d *Diff, path string, oldVal, newVal any) {
	if oldVal == newVal {
		return
	}
	d.Items = append(d.Items, DiffItem{
		Kind:     KindHeader,
		Category: CategoryModified,
		Path:     path,
		Old:      oldVal,
		New:      newVal,
		Message:  fmt.Sprintf("%s changed", path),
	})
}

func diffBlocks(a, b *lvrsrc.File, opts Options, d *Diff) {
	aBlocks := indexBlocks(a.Blocks)
	bBlocks := indexBlocks(b.Blocks)

	types := unionKeys(aBlocks, bBlocks)
	sort.Strings(types)

	for _, t := range types {
		ab, aok := aBlocks[t]
		bb, bok := bBlocks[t]
		switch {
		case !aok:
			d.Items = append(d.Items, DiffItem{
				Kind:     KindBlock,
				Category: CategoryAdded,
				Path:     "blocks." + t,
				New:      blockSummary(bb),
				Message:  fmt.Sprintf("resource type %q added", t),
			})
		case !bok:
			d.Items = append(d.Items, DiffItem{
				Kind:     KindBlock,
				Category: CategoryRemoved,
				Path:     "blocks." + t,
				Old:      blockSummary(ab),
				Message:  fmt.Sprintf("resource type %q removed", t),
			})
		default:
			diffSections(t, ab.Sections, bb.Sections, opts, d)
		}
	}
}

func indexBlocks(blocks []lvrsrc.Block) map[string]lvrsrc.Block {
	out := make(map[string]lvrsrc.Block, len(blocks))
	for _, b := range blocks {
		out[b.Type] = b
	}
	return out
}

func unionKeys(a, b map[string]lvrsrc.Block) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

type blockDigest struct {
	SectionCount int
	TotalSize    int
}

func blockSummary(b lvrsrc.Block) blockDigest {
	digest := blockDigest{SectionCount: len(b.Sections)}
	for _, s := range b.Sections {
		digest.TotalSize += len(s.Payload)
	}
	return digest
}

func diffSections(blockType string, a, b []lvrsrc.Section, opts Options, d *Diff) {
	aSec := indexSections(a)
	bSec := indexSections(b)

	indices := unionIndices(aSec, bSec)
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

	for _, idx := range indices {
		aS, aok := aSec[idx]
		bS, bok := bSec[idx]
		path := fmt.Sprintf("blocks.%s/%d", blockType, idx)
		switch {
		case !aok:
			d.Items = append(d.Items, DiffItem{
				Kind:     KindSection,
				Category: CategoryAdded,
				Path:     path,
				New:      sectionSummary(bS),
				Message:  fmt.Sprintf("section %s added", path),
			})
		case !bok:
			d.Items = append(d.Items, DiffItem{
				Kind:     KindSection,
				Category: CategoryRemoved,
				Path:     path,
				Old:      sectionSummary(aS),
				Message:  fmt.Sprintf("section %s removed", path),
			})
		default:
			diffSectionPair(blockType, path, aS, bS, opts, d)
		}
	}
}

func indexSections(sections []lvrsrc.Section) map[int32]lvrsrc.Section {
	out := make(map[int32]lvrsrc.Section, len(sections))
	for _, s := range sections {
		out[s.Index] = s
	}
	return out
}

func unionIndices(a, b map[int32]lvrsrc.Section) []int32 {
	seen := make(map[int32]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]int32, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

type sectionDigest struct {
	Name string
	Size int
	Hash string
}

func sectionSummary(s lvrsrc.Section) sectionDigest {
	return sectionDigest{
		Name: s.Name,
		Size: len(s.Payload),
		Hash: payloadHash(s.Payload),
	}
}

func payloadHash(p []byte) string {
	sum := sha256.Sum256(p)
	return hex.EncodeToString(sum[:])
}

func diffSectionPair(blockType, path string, a, b lvrsrc.Section, opts Options, d *Diff) {
	if a.Name != b.Name {
		d.Items = append(d.Items, DiffItem{
			Kind:     KindSection,
			Category: CategoryModified,
			Path:     path + ".name",
			Old:      a.Name,
			New:      b.Name,
			Message:  fmt.Sprintf("%s name changed", path),
		})
	}

	oldSize, newSize := len(a.Payload), len(b.Payload)
	if oldSize != newSize {
		d.Items = append(d.Items, DiffItem{
			Kind:     KindSection,
			Category: CategoryModified,
			Path:     path + ".size",
			Old:      oldSize,
			New:      newSize,
			Message:  fmt.Sprintf("%s payload size changed", path),
		})
	}

	contentChanged := false
	if oldSize == newSize {
		// Same size — check content via hash to detect in-place mutation.
		oldHash, newHash := payloadHash(a.Payload), payloadHash(b.Payload)
		if oldHash != newHash {
			contentChanged = true
			d.Items = append(d.Items, DiffItem{
				Kind:     KindSection,
				Category: CategoryModified,
				Path:     path + ".content",
				Old:      oldHash,
				New:      newHash,
				Message:  fmt.Sprintf("%s payload content changed", path),
			})
		}
	} else {
		// Size already flagged; also record hashes so consumers can see them.
		contentChanged = true
		d.Items = append(d.Items, DiffItem{
			Kind:     KindSection,
			Category: CategoryModified,
			Path:     path + ".content",
			Old:      payloadHash(a.Payload),
			New:      payloadHash(b.Payload),
			Message:  fmt.Sprintf("%s payload content changed", path),
		})
	}

	if !contentChanged {
		return
	}

	differ, ok := opts.DecodedDiffers[blockType]
	if !ok || differ == nil {
		return
	}
	d.Items = append(d.Items, differ(blockType, a.Index, a.Payload, b.Payload)...)
}
