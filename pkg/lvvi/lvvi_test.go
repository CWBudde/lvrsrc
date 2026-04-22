package lvvi

import (
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestVersionIsZero(t *testing.T) {
	cases := []struct {
		name string
		v    Version
		want bool
	}{
		{"zero value", Version{}, true},
		{"with format version", Version{FormatVersion: 3}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.IsZero(); got != tc.want {
				t.Fatalf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectVersionReturnsHeaderFormatVersion(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 3},
	}

	v, ok := DetectVersion(f)
	if !ok {
		t.Fatalf("DetectVersion ok = false, want true")
	}
	if v.FormatVersion != 3 {
		t.Fatalf("FormatVersion = %d, want 3", v.FormatVersion)
	}
}

func TestDetectVersionPassesThroughZeroFormatVersion(t *testing.T) {
	// A zero FormatVersion is still "what the file says". Detection succeeds
	// but the returned Version reports IsZero.
	f := &lvrsrc.File{Header: lvrsrc.Header{FormatVersion: 0}}

	v, ok := DetectVersion(f)
	if !ok {
		t.Fatalf("DetectVersion ok = false, want true")
	}
	if !v.IsZero() {
		t.Fatalf("Version = %+v, want zero", v)
	}
}

func TestDetectVersionReturnsFalseForNilFile(t *testing.T) {
	v, ok := DetectVersion(nil)
	if ok {
		t.Fatalf("DetectVersion(nil) ok = true, want false")
	}
	if !v.IsZero() {
		t.Fatalf("DetectVersion(nil) Version = %+v, want zero", v)
	}
}

func TestFileKindReExports(t *testing.T) {
	// Verify that the re-exported constants are interchangeable with the
	// pkg/lvrsrc originals.
	cases := []struct {
		name     string
		local    FileKind
		upstream lvrsrc.FileKind
	}{
		{"unknown", FileKindUnknown, lvrsrc.FileKindUnknown},
		{"vi", FileKindVI, lvrsrc.FileKindVI},
		{"control", FileKindControl, lvrsrc.FileKindControl},
		{"template", FileKindTemplate, lvrsrc.FileKindTemplate},
		{"library", FileKindLibrary, lvrsrc.FileKindLibrary},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.local != tc.upstream {
				t.Fatalf("%s: local = %q, upstream = %q", tc.name, tc.local, tc.upstream)
			}
		})
	}
}
