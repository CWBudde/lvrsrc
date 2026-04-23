package lvmeta

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func strgPayload(t *testing.T, text string) []byte {
	t.Helper()
	out := make([]byte, 4+len(text))
	binary.BigEndian.PutUint32(out[:4], uint32(len(text)))
	copy(out[4:], text)
	return out
}

// fileWithSingleSTRG returns a minimal *lvrsrc.File with a single STRG block,
// one section, plus one opaque "OPQ " block that later tests can assert is
// preserved byte-for-byte.
func fileWithSingleSTRG(t *testing.T, text string) *lvrsrc.File {
	t.Helper()
	return &lvrsrc.File{
		Header:  lvrsrc.Header{FormatVersion: 10},
		Kind:    lvrsrc.FileKindVI,
		RawTail: []byte{0x01, 0x02, 0x03, 0x04},
		Names: []lvrsrc.NameEntry{
			{Offset: 0, Value: "hello", Consumed: 6},
		},
		Blocks: []lvrsrc.Block{
			{
				Type: "OPQ ",
				Sections: []lvrsrc.Section{
					{Index: 1, DataOffset: 0x100, Payload: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
				},
			},
			{
				Type: "STRG",
				Sections: []lvrsrc.Section{
					{Index: 2, DataOffset: 0x200, Name: "desc", Payload: strgPayload(t, text)},
				},
			},
		},
	}
}

func TestApplyTypedEditSuccess(t *testing.T) {
	f := fileWithSingleSTRG(t, "old description")
	m := Mutator{}

	err := m.applyTypedEdit(f, strg.FourCC, func(v any) (any, error) {
		sv := v.(strg.Value)
		sv.Text = "new description"
		return sv, nil
	})
	if err != nil {
		t.Fatalf("applyTypedEdit returned %v, want nil", err)
	}

	got := string(f.Blocks[1].Sections[0].Payload[4:])
	if got != "new description" {
		t.Fatalf("payload text = %q, want %q", got, "new description")
	}
	size := binary.BigEndian.Uint32(f.Blocks[1].Sections[0].Payload[:4])
	if size != uint32(len("new description")) {
		t.Fatalf("payload size = %d, want %d", size, len("new description"))
	}
}

func TestApplyTypedEditPreservesUntouchedState(t *testing.T) {
	f := fileWithSingleSTRG(t, "old")
	opaquePayloadBefore := append([]byte(nil), f.Blocks[0].Sections[0].Payload...)
	rawTailBefore := append([]byte(nil), f.RawTail...)
	namesBefore := append([]lvrsrc.NameEntry(nil), f.Names...)

	err := Mutator{}.applyTypedEdit(f, strg.FourCC, func(v any) (any, error) {
		return strg.Value{Text: "new"}, nil
	})
	if err != nil {
		t.Fatalf("applyTypedEdit error = %v", err)
	}

	if got := f.Blocks[0].Sections[0].Payload; !bytesEqual(got, opaquePayloadBefore) {
		t.Fatalf("opaque payload changed: got %x, want %x", got, opaquePayloadBefore)
	}
	if !bytesEqual(f.RawTail, rawTailBefore) {
		t.Fatalf("RawTail changed: got %x, want %x", f.RawTail, rawTailBefore)
	}
	if len(f.Names) != len(namesBefore) || f.Names[0] != namesBefore[0] {
		t.Fatalf("Names changed: got %+v, want %+v", f.Names, namesBefore)
	}
	if f.Blocks[1].Sections[0].Index != 2 || f.Blocks[1].Sections[0].DataOffset != 0x200 {
		t.Fatalf("section metadata changed: %+v", f.Blocks[1].Sections[0])
	}
	if f.Blocks[1].Sections[0].Name != "desc" {
		t.Fatalf("section name changed: %q", f.Blocks[1].Sections[0].Name)
	}
}

func TestApplyTypedEditTargetMissing(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{{Type: "OPQ "}},
	}

	err := Mutator{}.applyTypedEdit(f, strg.FourCC, noopMutate)
	if !errors.Is(err, ErrTargetMissing) {
		t.Fatalf("err = %v, want ErrTargetMissing", err)
	}
	var me *MutationError
	if !errors.As(err, &me) {
		t.Fatalf("err is not *MutationError: %T", err)
	}
	if me.FourCC != strg.FourCC {
		t.Fatalf("MutationError.FourCC = %q, want %q", me.FourCC, strg.FourCC)
	}
}

func TestApplyTypedEditTargetAmbiguous(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{Type: "STRG", Sections: []lvrsrc.Section{
				{Index: 1, Payload: strgPayload(t, "a")},
				{Index: 2, Payload: strgPayload(t, "b")},
			}},
		},
	}

	err := Mutator{}.applyTypedEdit(f, strg.FourCC, noopMutate)
	if !errors.Is(err, ErrTargetAmbiguous) {
		t.Fatalf("err = %v, want ErrTargetAmbiguous", err)
	}
}

func TestApplyTypedEditUnsafeCodecTier(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{Type: "XXXX", Sections: []lvrsrc.Section{{Index: 1, Payload: []byte{0x00}}}},
		},
	}
	reg := codecs.New()
	reg.Register(fakeCodec{
		capability: codecs.Capability{
			FourCC:        "XXXX",
			WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
			Safety:        codecs.SafetyTier1,
		},
	})
	m := Mutator{registry: reg}

	err := m.applyTypedEdit(f, "XXXX", noopMutate)
	if !errors.Is(err, ErrUnsafeForTier2) {
		t.Fatalf("err = %v, want ErrUnsafeForTier2", err)
	}
}

func TestApplyTypedEditUnsupportedVersion(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 3},
		Blocks: []lvrsrc.Block{
			{Type: "XXXX", Sections: []lvrsrc.Section{{Index: 1, DataOffset: 0x42, Payload: []byte{0x00}}}},
		},
	}
	reg := codecs.New()
	reg.Register(fakeCodec{
		capability: codecs.Capability{
			FourCC:        "XXXX",
			WriteVersions: codecs.VersionRange{Min: 10, Max: 20},
			Safety:        codecs.SafetyTier2,
		},
	})
	m := Mutator{registry: reg}

	err := m.applyTypedEdit(f, "XXXX", noopMutate)
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("err = %v, want ErrUnsupportedVersion", err)
	}
	var me *MutationError
	if !errors.As(err, &me) {
		t.Fatalf("not MutationError: %T", err)
	}
	if me.Offset != 0x42 {
		t.Fatalf("MutationError.Offset = %d, want 0x42", me.Offset)
	}
}

func TestApplyTypedEditDecodeFailure(t *testing.T) {
	// STRG decode requires at least 4 bytes; supply 2.
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{Type: "STRG", Sections: []lvrsrc.Section{{Index: 1, Payload: []byte{0x00, 0x00}}}},
		},
	}

	err := Mutator{}.applyTypedEdit(f, strg.FourCC, noopMutate)
	if !errors.Is(err, ErrCodecDecode) {
		t.Fatalf("err = %v, want ErrCodecDecode", err)
	}
}

func TestApplyTypedEditMutationCallbackError(t *testing.T) {
	f := fileWithSingleSTRG(t, "orig")
	orig := append([]byte(nil), f.Blocks[1].Sections[0].Payload...)

	sentinel := errors.New("boom")
	err := Mutator{}.applyTypedEdit(f, strg.FourCC, func(v any) (any, error) {
		return nil, sentinel
	})
	if !errors.Is(err, ErrMutation) {
		t.Fatalf("err = %v, want ErrMutation", err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, should wrap sentinel", err)
	}
	if !bytesEqual(f.Blocks[1].Sections[0].Payload, orig) {
		t.Fatalf("payload mutated despite failure")
	}
}

func TestApplyTypedEditEncodeFailure(t *testing.T) {
	f := fileWithSingleSTRG(t, "orig")
	orig := append([]byte(nil), f.Blocks[1].Sections[0].Payload...)

	// Returning an incompatible type makes strg.Encode error out.
	err := Mutator{}.applyTypedEdit(f, strg.FourCC, func(v any) (any, error) {
		return 42, nil
	})
	if !errors.Is(err, ErrCodecEncode) {
		t.Fatalf("err = %v, want ErrCodecEncode", err)
	}
	if !bytesEqual(f.Blocks[1].Sections[0].Payload, orig) {
		t.Fatalf("payload mutated despite encode failure")
	}
}

func TestApplyTypedEditPostEditValidationError(t *testing.T) {
	// fakeCodec.Encode returns a payload that Validate reports as a
	// severity-error issue.
	reg := codecs.New()
	reg.Register(fakeCodec{
		capability: codecs.Capability{
			FourCC:        "XXXX",
			WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
			Safety:        codecs.SafetyTier2,
		},
		encode: func(codecs.Context, any) ([]byte, error) { return []byte("bad"), nil },
		decode: func(codecs.Context, []byte) (any, error) { return "ok", nil },
		validate: func(ctx codecs.Context, payload []byte) []validate.Issue {
			if string(payload) == "bad" {
				return []validate.Issue{{Severity: validate.SeverityError, Code: "x.bad"}}
			}
			return nil
		},
	})
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 0},
		Blocks: []lvrsrc.Block{
			{Type: "XXXX", Sections: []lvrsrc.Section{{Index: 1, Payload: []byte("good")}}},
		},
	}
	orig := append([]byte(nil), f.Blocks[0].Sections[0].Payload...)

	m := Mutator{registry: reg}
	err := m.applyTypedEdit(f, "XXXX", func(v any) (any, error) { return v, nil })
	if !errors.Is(err, ErrPostEditValidation) {
		t.Fatalf("err = %v, want ErrPostEditValidation", err)
	}
	if !bytesEqual(f.Blocks[0].Sections[0].Payload, orig) {
		t.Fatalf("payload replaced despite post-edit validation error")
	}
}

func TestApplyTypedEditStrictRejectsNewWarning(t *testing.T) {
	reg := codecs.New()
	var encodeOutput []byte
	reg.Register(fakeCodec{
		capability: codecs.Capability{
			FourCC:        "XXXX",
			WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
			Safety:        codecs.SafetyTier2,
		},
		decode: func(codecs.Context, []byte) (any, error) { return "", nil },
		encode: func(codecs.Context, any) ([]byte, error) { return encodeOutput, nil },
		validate: func(ctx codecs.Context, payload []byte) []validate.Issue {
			if string(payload) == "warnme" {
				return []validate.Issue{{Severity: validate.SeverityWarning, Code: "x.warn"}}
			}
			return nil
		},
	})
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 0},
		Blocks: []lvrsrc.Block{
			{Type: "XXXX", Sections: []lvrsrc.Section{{Index: 1, Payload: []byte("ok")}}},
		},
	}
	encodeOutput = []byte("warnme")

	// Strict mode: new warning fails.
	strict := Mutator{Strict: true, registry: reg}
	err := strict.applyTypedEdit(f, "XXXX", func(v any) (any, error) { return v, nil })
	if !errors.Is(err, ErrPostEditWarning) {
		t.Fatalf("strict err = %v, want ErrPostEditWarning", err)
	}
	if string(f.Blocks[0].Sections[0].Payload) != "ok" {
		t.Fatalf("payload replaced despite strict warning failure: %q", f.Blocks[0].Sections[0].Payload)
	}

	// Lenient mode: new warning is tolerated.
	lenient := Mutator{Strict: false, registry: reg}
	if err := lenient.applyTypedEdit(f, "XXXX", func(v any) (any, error) { return v, nil }); err != nil {
		t.Fatalf("lenient err = %v, want nil", err)
	}
	if string(f.Blocks[0].Sections[0].Payload) != "warnme" {
		t.Fatalf("lenient payload = %q, want warnme", f.Blocks[0].Sections[0].Payload)
	}
}

func TestApplyTypedEditStrictAllowsPreExistingWarning(t *testing.T) {
	// Pre-edit payload already emits a warning. After edit, same warning
	// re-appears: strict mode must tolerate it because the code isn't new.
	reg := codecs.New()
	reg.Register(fakeCodec{
		capability: codecs.Capability{
			FourCC:        "XXXX",
			WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
			Safety:        codecs.SafetyTier2,
		},
		decode: func(codecs.Context, []byte) (any, error) { return "", nil },
		encode: func(codecs.Context, any) ([]byte, error) { return []byte("warny-new"), nil },
		validate: func(ctx codecs.Context, payload []byte) []validate.Issue {
			if strings.HasPrefix(string(payload), "warny") {
				return []validate.Issue{{Severity: validate.SeverityWarning, Code: "x.warn"}}
			}
			return nil
		},
	})
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 0},
		Blocks: []lvrsrc.Block{
			{Type: "XXXX", Sections: []lvrsrc.Section{{Index: 1, Payload: []byte("warny-old")}}},
		},
	}

	m := Mutator{Strict: true, registry: reg}
	if err := m.applyTypedEdit(f, "XXXX", func(v any) (any, error) { return v, nil }); err != nil {
		t.Fatalf("strict err = %v, want nil", err)
	}
	if string(f.Blocks[0].Sections[0].Payload) != "warny-new" {
		t.Fatalf("payload = %q, want warny-new", f.Blocks[0].Sections[0].Payload)
	}
}

// --- helpers --------------------------------------------------------------

func noopMutate(v any) (any, error) { return v, nil }

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type fakeCodec struct {
	capability codecs.Capability
	decode     func(codecs.Context, []byte) (any, error)
	encode     func(codecs.Context, any) ([]byte, error)
	validate   func(codecs.Context, []byte) []validate.Issue
}

func (fc fakeCodec) Capability() codecs.Capability { return fc.capability }

func (fc fakeCodec) Decode(ctx codecs.Context, payload []byte) (any, error) {
	if fc.decode == nil {
		return nil, fmt.Errorf("fake: no decode")
	}
	return fc.decode(ctx, payload)
}

func (fc fakeCodec) Encode(ctx codecs.Context, value any) ([]byte, error) {
	if fc.encode == nil {
		return nil, fmt.Errorf("fake: no encode")
	}
	return fc.encode(ctx, value)
}

func (fc fakeCodec) Validate(ctx codecs.Context, payload []byte) []validate.Issue {
	if fc.validate == nil {
		return nil
	}
	return fc.validate(ctx, payload)
}
