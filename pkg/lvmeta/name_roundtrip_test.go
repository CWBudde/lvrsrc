package lvmeta

import (
	"bytes"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// TestSetNameRoundTripCorpusLVSR verifies that SetName leaves the container
// in a state that Serialize + re-Parse agree on, and that the validator does
// not flag the edited file. This exercises the "serializer and validator stay
// in sync" requirement from PLAN 4.4.4.
func TestSetNameRoundTripCorpusLVSR(t *testing.T) {
	path := corpus.Path("format-string.vi")

	f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s) = %v", path, err)
	}

	const newName = "renamed.vi"
	if err := SetName(f, newName); err != nil {
		t.Fatalf("SetName err = %v", err)
	}

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo err = %v", err)
	}

	round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("re-Parse err = %v", err)
	}

	// The LVSR section should surface the new name after round-trip.
	var foundLVSR bool
	for _, b := range round.Blocks {
		if b.Type != lvsrFourCC {
			continue
		}
		foundLVSR = true
		if len(b.Sections) == 0 {
			t.Fatalf("round-tripped LVSR block has no sections")
		}
		if got := b.Sections[0].Name; got != newName {
			t.Fatalf("round-tripped LVSR Name = %q, want %q", got, newName)
		}
	}
	if !foundLVSR {
		t.Fatalf("no LVSR block in round-tripped file")
	}

	// Validator should produce no new errors (round-tripped corpus files
	// are pre-validated by the existing suite; a SetName edit must not
	// regress them).
	for _, iss := range round.Validate() {
		if iss.Severity == lvrsrc.SeverityError {
			t.Fatalf("post-rename validator reported error %s: %s", iss.Code, iss.Message)
		}
	}
}
