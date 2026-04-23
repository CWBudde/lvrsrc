// Package lvvi provides a higher-level LabVIEW VI model layered on top of
// the generic RSRC container in pkg/lvrsrc. It exposes version detection
// today and will grow decoded-resource accessors in later phases.
package lvvi

import (
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// FileKind is re-exported from pkg/lvrsrc so callers that only need the
// VI-level API do not have to import both packages.
type FileKind = lvrsrc.FileKind

const (
	FileKindUnknown  = lvrsrc.FileKindUnknown
	FileKindVI       = lvrsrc.FileKindVI
	FileKindControl  = lvrsrc.FileKindControl
	FileKindTemplate = lvrsrc.FileKindTemplate
	FileKindLibrary  = lvrsrc.FileKindLibrary
)

// Version captures what the toolkit knows about a LabVIEW file's version.
//
// FormatVersion always reflects the RSRC container header. The remaining
// fields mirror the decoded "vers" resource (see internal/codecs/vers) and
// are populated by DecodeKnownResources when a `vers` section is present.
// HasApp reports whether the application-version fields carry meaningful
// data; callers should check it before reading Major/Minor/Patch/Stage/
// Build/Text.
type Version struct {
	// FormatVersion is the RSRC container format version from the file
	// header. A value of 0 means the header did not carry a version.
	FormatVersion uint16

	// HasApp reports whether the application-version fields below were
	// populated from a decoded `vers` resource.
	HasApp bool

	// Major is the LabVIEW application major version (e.g. 25 for 25.x).
	Major uint8
	// Minor is the LabVIEW application minor version, stored in the high
	// nibble of the second byte of the `vers` payload.
	Minor uint8
	// Patch is the LabVIEW application patch version, stored in the low
	// nibble of the second byte of the `vers` payload.
	Patch uint8
	// Stage mirrors the `vers` stage byte (0x80 = release in all known
	// samples).
	Stage uint8
	// Build is the LabVIEW application build number.
	Build uint8
	// Text is the ASCII version label carried by the `vers` resource
	// (e.g. "25.1.2" or "25.0").
	Text string
}

// IsZero reports whether v carries no version information at all
// (neither a container format version nor decoded application data).
func (v Version) IsZero() bool { return v == Version{} }

// DetectVersion reads whatever version information is currently derivable
// from f. It returns (zero, false) if f is nil; otherwise it returns the
// populated Version and true.
//
// DetectVersion is a package function rather than a method on *lvrsrc.File
// because pkg/lvrsrc cannot import pkg/lvvi without creating a cycle.
func DetectVersion(f *lvrsrc.File) (Version, bool) {
	if f == nil {
		return Version{}, false
	}
	return Version{FormatVersion: f.Header.FormatVersion}, true
}
