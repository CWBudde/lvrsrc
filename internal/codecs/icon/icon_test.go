package icon

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestCapabilities(t *testing.T) {
	cases := []struct {
		name     string
		codec    codecs.ResourceCodec
		fourCC   string
		bits     int
		rawBytes int
	}{
		{"ICON", MonoCodec{}, "ICON", 1, 128},
		{"icl4", Color4Codec{}, "icl4", 4, 512},
		{"icl8", Color8Codec{}, "icl8", 8, 1024},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.codec.Capability()
			if c.FourCC != tc.fourCC {
				t.Fatalf("FourCC = %q, want %q", c.FourCC, tc.fourCC)
			}
			if c.Safety != codecs.SafetyTier2 {
				t.Fatalf("Safety = %v, want SafetyTier2", c.Safety)
			}
		})
	}
}

func TestDecodeEncodeRoundTripSynthetic(t *testing.T) {
	cases := []struct {
		name         string
		codec        codecs.ResourceCodec
		buildPayload func() []byte
		wantFirstRow []byte
		wantBits     int
		wantFourCC   string
	}{
		{
			name:  "ICON",
			codec: MonoCodec{},
			buildPayload: func() []byte {
				payload := make([]byte, 128)
				payload[0] = 0xAA
				payload[1] = 0x55
				payload[2] = 0xF0
				payload[3] = 0x0F
				return payload
			},
			wantFirstRow: []byte{
				1, 0, 1, 0, 1, 0, 1, 0,
				0, 1, 0, 1, 0, 1, 0, 1,
				1, 1, 1, 1, 0, 0, 0, 0,
				0, 0, 0, 0, 1, 1, 1, 1,
			},
			wantBits:   1,
			wantFourCC: "ICON",
		},
		{
			name:  "icl4",
			codec: Color4Codec{},
			buildPayload: func() []byte {
				payload := make([]byte, 512)
				payload[0] = 0x1F
				payload[1] = 0x20
				payload[2] = 0xAB
				payload[3] = 0xCD
				return payload
			},
			wantFirstRow: []byte{1, 15, 2, 0, 10, 11, 12, 13},
			wantBits:     4,
			wantFourCC:   "icl4",
		},
		{
			name:  "icl8",
			codec: Color8Codec{},
			buildPayload: func() []byte {
				payload := make([]byte, 1024)
				copy(payload[:8], []byte{0x00, 0x7F, 0x80, 0xFF, 0x12, 0x34, 0x56, 0x78})
				return payload
			},
			wantFirstRow: []byte{0x00, 0x7F, 0x80, 0xFF, 0x12, 0x34, 0x56, 0x78},
			wantBits:     8,
			wantFourCC:   "icl8",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := tc.buildPayload()
			got, err := tc.codec.Decode(codecs.Context{}, payload)
			if err != nil {
				t.Fatalf("Decode error = %v", err)
			}
			v := got.(Value)
			if v.FourCC != tc.wantFourCC {
				t.Fatalf("FourCC = %q, want %q", v.FourCC, tc.wantFourCC)
			}
			if v.Width != Width || v.Height != Height {
				t.Fatalf("dimensions = %dx%d, want %dx%d", v.Width, v.Height, Width, Height)
			}
			if v.BitsPerPixel != tc.wantBits {
				t.Fatalf("BitsPerPixel = %d, want %d", v.BitsPerPixel, tc.wantBits)
			}
			if got := v.Pixels[:len(tc.wantFirstRow)]; !bytes.Equal(got, tc.wantFirstRow) {
				t.Fatalf("first decoded pixels = %v, want %v", got, tc.wantFirstRow)
			}

			back, err := tc.codec.Encode(codecs.Context{}, v)
			if err != nil {
				t.Fatalf("Encode error = %v", err)
			}
			if !bytes.Equal(back, payload) {
				t.Fatalf("round-trip mismatch")
			}
		})
	}
}

func TestEncodeAcceptsPointer(t *testing.T) {
	v := Value{
		FourCC:       "icl8",
		Width:        Width,
		Height:       Height,
		BitsPerPixel: 8,
		Pixels:       make([]byte, PixelCount),
	}
	out, err := Color8Codec{}.Encode(codecs.Context{}, &v)
	if err != nil {
		t.Fatalf("Encode(*Value) error = %v", err)
	}
	if len(out) != 1024 {
		t.Fatalf("len(out) = %d, want 1024", len(out))
	}
}

func TestEncodeRejectsWrongPixelCount(t *testing.T) {
	_, err := MonoCodec{}.Encode(codecs.Context{}, Value{
		FourCC:       "ICON",
		Width:        Width,
		Height:       Height,
		BitsPerPixel: 1,
		Pixels:       make([]byte, PixelCount-1),
	})
	if err == nil {
		t.Fatalf("Encode with wrong pixel count did not error")
	}
}

func TestEncodeRejectsOutOfRangePixel(t *testing.T) {
	v := Value{
		FourCC:       "icl4",
		Width:        Width,
		Height:       Height,
		BitsPerPixel: 4,
		Pixels:       make([]byte, PixelCount),
	}
	v.Pixels[17] = 16
	_, err := Color4Codec{}.Encode(codecs.Context{}, v)
	if err == nil {
		t.Fatalf("Encode with out-of-range pixel did not error")
	}
}

func TestValidateReportsWrongSize(t *testing.T) {
	issues := MonoCodec{}.Validate(codecs.Context{}, make([]byte, 127))
	assertHasCode(t, issues, "icon.payload.size", validate.SeverityError)

	issues = Color4Codec{}.Validate(codecs.Context{}, make([]byte, 511))
	assertHasCode(t, issues, "icl4.payload.size", validate.SeverityError)

	issues = Color8Codec{}.Validate(codecs.Context{}, make([]byte, 1023))
	assertHasCode(t, issues, "icl8.payload.size", validate.SeverityError)
}

func TestCorpusRoundTrip(t *testing.T) {
	type testCase struct {
		fourCC string
		codec  codecs.ResourceCodec
	}
	cases := []testCase{
		{fourCC: "ICON", codec: MonoCodec{}},
		{fourCC: "icl4", codec: Color4Codec{}},
		{fourCC: "icl8", codec: Color8Codec{}},
	}

	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}

	total := 0
	for _, tc := range cases {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := filepath.Ext(e.Name())
			if ext != ".vi" && ext != ".ctl" {
				continue
			}
			path := filepath.Join(corpus.Dir(), e.Name())
			f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("open %s: %v", e.Name(), err)
			}
			for _, block := range f.Blocks {
				if block.Type != tc.fourCC {
					continue
				}
				for _, section := range block.Sections {
					total++
					got, err := tc.codec.Decode(codecs.Context{}, section.Payload)
					if err != nil {
						t.Fatalf("%s %s id=%d Decode: %v", e.Name(), tc.fourCC, section.Index, err)
					}
					back, err := tc.codec.Encode(codecs.Context{}, got)
					if err != nil {
						t.Fatalf("%s %s id=%d Encode: %v", e.Name(), tc.fourCC, section.Index, err)
					}
					if !bytes.Equal(back, section.Payload) {
						t.Fatalf("%s %s id=%d round-trip mismatch", e.Name(), tc.fourCC, section.Index)
					}
					if issues := tc.codec.Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
						t.Fatalf("%s %s id=%d Validate issues: %+v", e.Name(), tc.fourCC, section.Index, issues)
					}
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no icon sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d icon sections", total)
}

func assertHasCode(t *testing.T, issues []validate.Issue, code string, sev validate.Severity) {
	t.Helper()
	for _, i := range issues {
		if i.Code == code && i.Severity == sev {
			return
		}
	}
	t.Fatalf("expected issue code=%q severity=%q in: %+v", code, sev, issues)
}
