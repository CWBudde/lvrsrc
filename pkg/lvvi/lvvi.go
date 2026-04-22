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
// Only the RSRC container format version is available today; fields for the
// decoded LabVIEW application version (major/minor/bugfix/stage/build) will
// be added when codecs for version-carrying resources (e.g. "vers", "LVSR")
// land in Phase 4.3.
type Version struct {
	// FormatVersion is the RSRC container format version from the file
	// header. A value of 0 means the header did not carry a version.
	FormatVersion uint16
}

// IsZero reports whether v carries no version information.
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
