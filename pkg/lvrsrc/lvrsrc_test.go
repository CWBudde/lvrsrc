package lvrsrc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestParse(t *testing.T) {
	data := readFixture(t, "config-data.ctl")

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got, want := f.Kind, lvrsrc.FileKindControl; got != want {
		t.Fatalf("Kind = %v, want %v", got, want)
	}

	if got, want := f.Header.Type, "LVCC"; got != want {
		t.Fatalf("Header.Type = %q, want %q", got, want)
	}

	if got, want := len(f.Blocks), 24; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}

	resources := f.Resources()
	if got, want := len(resources), 28; got != want {
		t.Fatalf("len(Resources()) = %d, want %d", got, want)
	}

	first := resources[0]
	if got, want := first.Type, "LIBN"; got != want {
		t.Fatalf("first.Type = %q, want %q", got, want)
	}
	if got, want := first.ID, int32(0); got != want {
		t.Fatalf("first.ID = %d, want %d", got, want)
	}
	if got, want := first.Size, len(f.Blocks[0].Sections[0].Payload); got != want {
		t.Fatalf("first.Size = %d, want %d", got, want)
	}

	if got, want := resources[1].Name, "Config Data.ctl"; got != want {
		t.Fatalf("resources[1].Name = %q, want %q", got, want)
	}

	if got, want := f.Names[0].Value, "Config Data.ctl"; got != want {
		t.Fatalf("Names[0].Value = %q, want %q", got, want)
	}
}

func TestOpen(t *testing.T) {
	path := fixturePath(t, "config-data.ctl")

	f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if got, want := f.Names[0].Value, "Config Data.ctl"; got != want {
		t.Fatalf("first name = %q, want %q", got, want)
	}
}

func TestCloneDeepCopy(t *testing.T) {
	data := readFixture(t, "config-data.ctl")

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	clone := f.Clone()
	if clone == f {
		t.Fatal("Clone() returned same pointer")
	}

	origType := f.Header.Type
	origName := f.Names[0].Value
	origSectionName := f.Blocks[0].Sections[0].Name
	origPayloadFirst := f.Blocks[0].Sections[0].Payload[0]

	clone.Header.Type = "TEST"
	clone.Names[0].Value = "mutated"
	clone.Blocks[0].Sections[0].Name = "changed"
	clone.Blocks[0].Sections[0].Payload[0] ^= 0xff

	if got := f.Header.Type; got != origType {
		t.Fatalf("original Header.Type mutated to %q", got)
	}
	if got := f.Names[0].Value; got != origName {
		t.Fatalf("original Names[0].Value mutated to %q", got)
	}
	if got := f.Blocks[0].Sections[0].Name; got != origSectionName {
		t.Fatalf("original section name mutated to %q", got)
	}
	if got := f.Blocks[0].Sections[0].Payload[0]; got != origPayloadFirst {
		t.Fatalf("original payload mutated to 0x%x", got)
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t, name))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", name, err)
	}
	return data
}
