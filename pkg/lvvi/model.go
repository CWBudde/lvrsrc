package lvvi

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdex"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdpw"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdse"
	"github.com/CWBudde/lvrsrc/internal/codecs/dthp"
	"github.com/CWBudde/lvrsrc/internal/codecs/fpex"
	"github.com/CWBudde/lvrsrc/internal/codecs/fpse"
	"github.com/CWBudde/lvrsrc/internal/codecs/ftab"
	"github.com/CWBudde/lvrsrc/internal/codecs/hist"
	"github.com/CWBudde/lvrsrc/internal/codecs/libn"
	"github.com/CWBudde/lvrsrc/internal/codecs/lvsr"
	"github.com/CWBudde/lvrsrc/internal/codecs/muid"
	"github.com/CWBudde/lvrsrc/internal/codecs/rtsg"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/codecs/vers"
	"github.com/CWBudde/lvrsrc/internal/codecs/vits"
	"github.com/CWBudde/lvrsrc/internal/codecs/vpdp"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// Issue is the public validation issue surfaced by higher-level decoding.
// It mirrors pkg/lvrsrc.Issue so callers don't have to import both packages
// when handling decode-time diagnostics.
type Issue = lvrsrc.Issue

// Severity mirrors pkg/lvrsrc.Severity.
type Severity = lvrsrc.Severity

const (
	// SeverityWarning mirrors lvrsrc.SeverityWarning.
	SeverityWarning = lvrsrc.SeverityWarning
	// SeverityError mirrors lvrsrc.SeverityError.
	SeverityError = lvrsrc.SeverityError
)

// ResourceSummary is a compact, stable description of a single RSRC
// section suitable for user-facing listings. One ResourceSummary is
// produced per section, not per block.
type ResourceSummary struct {
	// FourCC is the block Type that owns the section.
	FourCC string
	// SectionID is the section's Index field.
	SectionID int32
	// Name is the section's surface name (from the container name table,
	// or empty if the section has no name).
	Name string
	// Size is the raw payload length in bytes.
	Size int
	// Decoded reports whether a typed codec (non-opaque) is registered
	// for FourCC.
	Decoded bool
}

// LVSRFlags is the decoded set of boolean settings from the VI's LVSR
// block. Fields are populated from the raw flag words via the codec in
// internal/codecs/lvsr; each field is documented there.
//
// All fields default to false; Model.Flags returns ok=false when the
// wrapped file has no LVSR block.
type LVSRFlags struct {
	SuspendOnRun      bool
	Locked            bool
	RunOnOpen         bool
	SavedForPrevious  bool
	SeparateCode      bool
	ClearIndicators   bool
	AutoErrorHandling bool
	HasBreakpoints    bool
	Debuggable        bool
}

// Model is the higher-level, read-oriented view of a LabVIEW file.
// It wraps a parsed *lvrsrc.File and caches values decoded from known
// resources (application version, description, LVSR flags).
//
// Model is read-only. For Tier 2 mutations use pkg/lvmeta.
type Model struct {
	file             *lvrsrc.File
	version          Version
	description      string
	hasDesc          bool
	lvsr             lvsr.Value
	hasLvsr          bool
	breakpointCount  uint32
	hasBreakpointCnt bool
}

// File returns the underlying parsed file. It is the same pointer passed
// to DecodeKnownResources; callers who want to mutate the file should go
// through pkg/lvmeta APIs.
func (m *Model) File() *lvrsrc.File {
	if m == nil {
		return nil
	}
	return m.file
}

// Description returns the decoded VI description and true when a `STRG`
// section was successfully decoded. It returns ("", false) when no STRG
// section exists or decoding failed (in which case DecodeKnownResources
// recorded an Issue).
func (m *Model) Description() (string, bool) {
	if m == nil || !m.hasDesc {
		return "", false
	}
	return m.description, true
}

// Version returns the cached Version and true when the model wraps a
// non-nil file; false when the receiver or the wrapped file is nil.
// The returned Version always carries FormatVersion from the container
// header; its HasApp field reports whether decoded application-version
// data (from a `vers` resource) is also present.
func (m *Model) Version() (Version, bool) {
	if m == nil || m.file == nil {
		return Version{}, false
	}
	return m.version, true
}

// Flags returns the decoded LVSR flag set and true when a valid LVSR
// section was decoded from the wrapped file. It returns a zero LVSRFlags
// and false when no LVSR section exists or decoding failed (in which case
// DecodeKnownResources recorded an Issue).
func (m *Model) Flags() (LVSRFlags, bool) {
	if m == nil || !m.hasLvsr {
		return LVSRFlags{}, false
	}
	return LVSRFlags{
		SuspendOnRun:      m.lvsr.SuspendOnRun(),
		Locked:            m.lvsr.Locked(),
		RunOnOpen:         m.lvsr.RunOnOpen(),
		SavedForPrevious:  m.lvsr.SavedForPrevious(),
		SeparateCode:      m.lvsr.SeparateCode(),
		ClearIndicators:   m.lvsr.ClearIndicators(),
		AutoErrorHandling: m.lvsr.AutoErrorHandling(),
		HasBreakpoints:    m.lvsr.HasBreakpoints(),
		Debuggable:        m.lvsr.Debuggable(),
	}, true
}

// BreakpointCount returns the integer stored at flag-word index 28 of
// the LVSR block. It returns ok=false when no LVSR section was decoded
// or the payload is too short to reach that word.
func (m *Model) BreakpointCount() (uint32, bool) {
	if m == nil || !m.hasLvsr || !m.hasBreakpointCnt {
		return 0, false
	}
	return m.breakpointCount, true
}

// ListResources returns one summary per section across all blocks, in
// the file's native block/section order. The slice is freshly allocated
// on each call; callers may retain it safely.
func (m *Model) ListResources() []ResourceSummary {
	if m == nil || m.file == nil {
		return nil
	}

	reg := newLvviRegistry()

	total := 0
	for _, b := range m.file.Blocks {
		total += len(b.Sections)
	}
	out := make([]ResourceSummary, 0, total)
	for _, b := range m.file.Blocks {
		decoded := reg.Has(b.Type)
		for _, s := range b.Sections {
			out = append(out, ResourceSummary{
				FourCC:    b.Type,
				SectionID: s.Index,
				Name:      s.Name,
				Size:      len(s.Payload),
				Decoded:   decoded,
			})
		}
	}
	return out
}

// DecodeKnownResources parses the known typed resources in f (currently
// `vers` for application version and `STRG` for the VI description),
// producing a Model that caches those values. Any per-resource decode
// failures are returned as Issues; the Model is returned regardless so
// callers can still access ListResources and FormatVersion.
//
// A nil f returns (nil, nil).
func DecodeKnownResources(f *lvrsrc.File) (*Model, []Issue) {
	if f == nil {
		return nil, nil
	}

	m := &Model{
		file:    f,
		version: Version{FormatVersion: f.Header.FormatVersion},
	}

	var issues []Issue
	reg := newLvviRegistry()
	ctx := codecs.Context{
		FileVersion: f.Header.FormatVersion,
		Kind:        f.Kind,
	}

	// Decode vers (first occurrence wins; extras are flagged).
	if refs := sectionsOf(f, vers.FourCC); len(refs) > 0 {
		issues = appendIfMultiple(issues, vers.FourCC, len(refs))
		payload := refs[0].Payload
		codec := reg.Lookup(vers.FourCC)
		raw, err := codec.Decode(ctx, payload)
		if err != nil {
			issues = append(issues, decodeErrorIssue(vers.FourCC, refs[0].Index, err))
		} else if v, ok := raw.(vers.Value); ok {
			m.version.HasApp = true
			m.version.Major = v.Major
			m.version.Minor = v.Minor
			m.version.Patch = v.Patch
			m.version.Stage = v.Stage
			m.version.Build = v.Build
			m.version.Text = v.Text
		}
	}

	// Decode STRG.
	if refs := sectionsOf(f, strg.FourCC); len(refs) > 0 {
		issues = appendIfMultiple(issues, strg.FourCC, len(refs))
		payload := refs[0].Payload
		codec := reg.Lookup(strg.FourCC)
		raw, err := codec.Decode(ctx, payload)
		if err != nil {
			issues = append(issues, decodeErrorIssue(strg.FourCC, refs[0].Index, err))
		} else if s, ok := raw.(strg.Value); ok {
			m.hasDesc = true
			m.description = s.Text
		}
	}

	// Decode LVSR (first occurrence wins; extras are flagged).
	if refs := sectionsOf(f, lvsr.FourCC); len(refs) > 0 {
		issues = appendIfMultiple(issues, lvsr.FourCC, len(refs))
		payload := refs[0].Payload
		codec := reg.Lookup(lvsr.FourCC)
		raw, err := codec.Decode(ctx, payload)
		if err != nil {
			issues = append(issues, decodeErrorIssue(lvsr.FourCC, refs[0].Index, err))
		} else if lv, ok := raw.(lvsr.Value); ok {
			m.hasLvsr = true
			m.lvsr = lv
			if n, ok := lv.BreakpointCount(); ok {
				m.breakpointCount = n
				m.hasBreakpointCnt = true
			}
		}
	}

	return m, issues
}

// newLvviRegistry builds a fresh registry populated with every codec the
// higher-level model knows how to decode. It mirrors the set in
// pkg/lvmeta so the two packages agree on which resources are "known".
func newLvviRegistry() *codecs.Registry {
	r := codecs.New()
	r.Register(strg.Codec{})
	r.Register(vers.Codec{})
	r.Register(lvsr.Codec{})
	r.Register(muid.Codec{})
	r.Register(fpse.Codec{})
	r.Register(bdse.Codec{})
	r.Register(vpdp.Codec{})
	r.Register(dthp.Codec{})
	r.Register(rtsg.Codec{})
	r.Register(libn.Codec{})
	r.Register(hist.Codec{})
	r.Register(bdpw.Codec{})
	r.Register(fpex.Codec{})
	r.Register(bdex.Codec{})
	r.Register(ftab.Codec{})
	r.Register(vits.Codec{})
	return r
}

// sectionsOf returns every section across every block of type fourCC, in
// the file's native order.
func sectionsOf(f *lvrsrc.File, fourCC string) []lvrsrc.Section {
	var out []lvrsrc.Section
	for _, b := range f.Blocks {
		if b.Type != fourCC {
			continue
		}
		out = append(out, b.Sections...)
	}
	return out
}

func decodeErrorIssue(fourCC string, sectionIndex int32, err error) Issue {
	return Issue{
		Severity: SeverityError,
		Code:     "lvvi.decode.failed",
		Message:  fmt.Sprintf("decode %q section %d: %v", fourCC, sectionIndex, err),
		Location: lvrsrc.IssueLocation{
			Area:         "resource",
			BlockType:    fourCC,
			SectionIndex: sectionIndex,
		},
	}
}

func appendIfMultiple(issues []Issue, fourCC string, count int) []Issue {
	if count <= 1 {
		return issues
	}
	return append(issues, Issue{
		Severity: SeverityWarning,
		Code:     "lvvi.decode.multiple_sections",
		Message:  fmt.Sprintf("found %d %q sections; model uses the first", count, fourCC),
		Location: lvrsrc.IssueLocation{Area: "resource", BlockType: fourCC},
	})
}
