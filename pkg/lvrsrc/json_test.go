package lvrsrc_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestMarshalJSONParsedFile(t *testing.T) {
	data := readFixture(t, "config-data.ctl")

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	blob, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got struct {
		Kind        string `json:"kind"`
		Compression string `json:"compression"`
		Header      struct {
			Type string `json:"type"`
		} `json:"header"`
		Blocks []struct {
			Type     string `json:"type"`
			Sections []struct {
				Index   int32  `json:"index"`
				Payload string `json:"payload"`
			} `json:"sections"`
		} `json:"blocks"`
		Names []struct {
			Value string `json:"value"`
		} `json:"names"`
	}
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(blob, &raw); err != nil {
		t.Fatalf("json.Unmarshal(raw) error = %v", err)
	}
	if _, ok := raw["header"]; !ok {
		t.Fatalf("top-level JSON missing %q key: %s", "header", string(blob))
	}
	if _, ok := raw["Header"]; ok {
		t.Fatalf("top-level JSON should not expose Go field name %q: %s", "Header", string(blob))
	}
	if _, ok := raw["blockInfoList"]; !ok {
		t.Fatalf("top-level JSON missing %q key: %s", "blockInfoList", string(blob))
	}
	if _, ok := raw["BlockInfoList"]; ok {
		t.Fatalf("top-level JSON should not expose Go field name %q: %s", "BlockInfoList", string(blob))
	}

	if got.Kind != string(lvrsrc.FileKindControl) {
		t.Fatalf("kind = %q, want %q", got.Kind, lvrsrc.FileKindControl)
	}
	if got.Compression != string(lvrsrc.CompressionKindUnknown) {
		t.Fatalf("compression = %q, want %q", got.Compression, lvrsrc.CompressionKindUnknown)
	}
	if got.Header.Type != "LVCC" {
		t.Fatalf("header.type = %q, want %q", got.Header.Type, "LVCC")
	}
	if len(got.Blocks) != 24 {
		t.Fatalf("len(blocks) = %d, want %d", len(got.Blocks), 24)
	}
	if got.Blocks[0].Type != "LIBN" {
		t.Fatalf("blocks[0].type = %q, want %q", got.Blocks[0].Type, "LIBN")
	}
	if got.Blocks[0].Sections[0].Index != 0 {
		t.Fatalf("blocks[0].sections[0].index = %d, want 0", got.Blocks[0].Sections[0].Index)
	}
	if len(got.Names) != 1 || got.Names[0].Value != "Config Data.ctl" {
		t.Fatalf("names = %+v, want single Config Data.ctl entry", got.Names)
	}

	wantPayload := base64.StdEncoding.EncodeToString(f.Blocks[0].Sections[0].Payload)
	if got.Blocks[0].Sections[0].Payload != wantPayload {
		t.Fatalf("payload = %q, want %q", got.Blocks[0].Sections[0].Payload, wantPayload)
	}
}

func TestMarshalJSONBase64OpaqueBytes(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{
			Magic: "RSRC\r\n",
			Type:  "LVIN",
		},
		Blocks: []lvrsrc.Block{
			{
				Type: "TEST",
				Sections: []lvrsrc.Section{
					{
						Index:   7,
						Name:    "opaque",
						Payload: []byte{0x00, 0xff, 0x10},
					},
				},
			},
		},
		RawTail: []byte{0xde, 0xad, 0xbe},
	}

	blob, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got struct {
		Blocks []struct {
			Sections []struct {
				Payload string `json:"payload"`
			} `json:"sections"`
		} `json:"blocks"`
		RawTail string `json:"rawTail"`
	}
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(blob, &raw); err != nil {
		t.Fatalf("json.Unmarshal(raw) error = %v", err)
	}
	blocks, ok := raw["blocks"].([]any)
	if !ok || len(blocks) != 1 {
		t.Fatalf("blocks JSON shape = %#v, want one block", raw["blocks"])
	}
	firstBlock, ok := blocks[0].(map[string]any)
	if !ok {
		t.Fatalf("first block JSON shape = %#v", blocks[0])
	}
	if gotCount, ok := firstBlock["sectionCount"].(float64); !ok || gotCount != 1 {
		t.Fatalf("sectionCount = %#v, want 1", firstBlock["sectionCount"])
	}
	if _, ok := firstBlock["SectionCountMinusOne"]; ok {
		t.Fatalf("block JSON should not expose %q: %s", "SectionCountMinusOne", string(blob))
	}

	if want := base64.StdEncoding.EncodeToString([]byte{0x00, 0xff, 0x10}); got.Blocks[0].Sections[0].Payload != want {
		t.Fatalf("payload = %q, want %q", got.Blocks[0].Sections[0].Payload, want)
	}
	if want := base64.StdEncoding.EncodeToString([]byte{0xde, 0xad, 0xbe}); got.RawTail != want {
		t.Fatalf("rawTail = %q, want %q", got.RawTail, want)
	}
}
