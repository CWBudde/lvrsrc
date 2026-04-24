//go:build js && wasm

// Package main provides a WebAssembly entry point for the lvrsrc web demo.
//
// The demo focuses on user-facing views (icon, version, description, link
// metadata, container schema, typed resource list), not on raw-byte tools. It
// exposes a single JS-callable handler, parseVI, that returns a pre-decoded
// bundle ready for the UI.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	iconcodec "github.com/CWBudde/lvrsrc/internal/codecs/icon"
	"github.com/CWBudde/lvrsrc/internal/codecs/libd"
	"github.com/CWBudde/lvrsrc/internal/codecs/lifp"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/codecs/vers"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

var wasmHandlers []js.Func

// WASMResult is returned as a JSON string to JavaScript.
type WASMResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// WASMParseData is the web-friendly representation of a parsed RSRC/VI file.
type WASMParseData struct {
	Kind            string         `json:"kind"`
	Compression     string         `json:"compression"`
	Summary         WASMSummary    `json:"summary"`
	Header          WASMHeader     `json:"header"`
	SecondaryHeader WASMHeader     `json:"secondary_header"`
	Resources       []WASMResource `json:"resources"`
	Info            WASMInfo       `json:"info"`
}

type WASMSummary struct {
	BlockCount         int `json:"block_count"`
	ResourceCount      int `json:"resource_count"`
	NamedResourceCount int `json:"named_resource_count"`
	NameCount          int `json:"name_count"`
	TotalPayloadBytes  int `json:"total_payload_bytes"`
	DecodedCount       int `json:"decoded_count"`
}

// WASMHeader is a JSON-friendly file header.
type WASMHeader struct {
	Magic         string `json:"magic"`
	FormatVersion uint16 `json:"format_version"`
	Type          string `json:"type"`
	Creator       string `json:"creator"`
	InfoOffset    uint32 `json:"info_offset"`
	InfoSize      uint32 `json:"info_size"`
	DataOffset    uint32 `json:"data_offset"`
	DataSize      uint32 `json:"data_size"`
}

// WASMResource is a single resource section.
type WASMResource struct {
	Type    string `json:"type"`
	ID      int32  `json:"id"`
	Name    string `json:"name,omitempty"`
	Size    int    `json:"size"`
	Decoded bool   `json:"decoded"`
}

// WASMInfo is the decoded user-facing metadata.
type WASMInfo struct {
	DisplayName string         `json:"display_name,omitempty"`
	Version     string         `json:"version,omitempty"`
	Description string         `json:"description,omitempty"`
	HasDesc     bool           `json:"has_desc"`
	Icon        *WASMIcon      `json:"icon,omitempty"`
	Deps        WASMDeps       `json:"deps"`
}

// WASMIcon carries the 32x32 VI icon pre-expanded into row-major RGBA bytes
// so the browser can paint it directly without palette lookups. The best
// available variant is selected server-side: icl8 > icl4 > ICON.
type WASMIcon struct {
	FourCC string `json:"fourcc"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	RGBA   string `json:"rgba"` // base64-encoded Width*Height*4 RGBA bytes
}

// WASMDeps groups decoded link-info entries by source.
type WASMDeps struct {
	FrontPanel   []WASMDepEntry `json:"front_panel"`
	BlockDiagram []WASMDepEntry `json:"block_diagram"`
}

// WASMDepEntry is one decoded link-info reference.
type WASMDepEntry struct {
	LinkType   string   `json:"link_type"`
	Qualifiers []string `json:"qualifiers"`
}

func main() {
	registerHandler("parseVI", handleParse)
	select {}
}

func registerHandler(name string, handler func(args []js.Value) (any, error)) {
	fn := js.FuncOf(func(_ js.Value, args []js.Value) (out any) {
		defer func() {
			if rec := recover(); rec != nil {
				out = errorResult(fmt.Sprintf("handler panicked: %v", rec))
			}
		}()

		data, err := handler(args)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(data)
	})
	wasmHandlers = append(wasmHandlers, fn)
	js.Global().Set(name, fn)
}

func handleParse(args []js.Value) (any, error) {
	file, err := parseFileArg(args, 0)
	if err != nil {
		return nil, err
	}
	resources := buildResources(file)

	result := WASMParseData{
		Kind:            kindName(file.Kind),
		Compression:     string(file.Compression),
		Summary:         buildSummary(file, resources),
		Header:          headerToWASM(file.Header),
		SecondaryHeader: headerToWASM(file.SecondaryHeader),
		Resources:       resources,
		Info:            buildInfo(file),
	}
	return result, nil
}

func parseFileArg(args []js.Value, idx int) (*lvrsrc.File, error) {
	if len(args) <= idx {
		return nil, fmt.Errorf("missing input data at argument %d", idx)
	}
	data := readBytesArg(args[idx])
	file, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	return file, nil
}

func readBytesArg(v js.Value) []byte {
	length := v.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, v)
	return data
}

// typedFourCCs lists FourCCs handled by a typed (non-opaque) codec. The web
// demo uses this only to flag rows in the resource list; the source of truth
// is internal/codecs/registry.go.
var typedFourCCs = map[string]struct{}{
	"vers": {},
	"STRG": {},
	"ICON": {},
	"icl4": {},
	"icl8": {},
	"CONP": {},
	"CPC2": {},
	"LIfp": {},
	"LIbd": {},
	"VCTP": {},
}

func buildResources(file *lvrsrc.File) []WASMResource {
	out := make([]WASMResource, 0)
	for _, block := range file.Blocks {
		_, typed := typedFourCCs[block.Type]
		for _, section := range block.Sections {
			out = append(out, WASMResource{
				Type:    block.Type,
				ID:      section.Index,
				Name:    section.Name,
				Size:    len(section.Payload),
				Decoded: typed,
			})
		}
	}
	return out
}

func buildSummary(file *lvrsrc.File, resources []WASMResource) WASMSummary {
	named := 0
	total := 0
	decoded := 0
	for _, r := range resources {
		if r.Name != "" {
			named++
		}
		total += r.Size
		if r.Decoded {
			decoded++
		}
	}
	return WASMSummary{
		BlockCount:         len(file.Blocks),
		ResourceCount:      len(resources),
		NamedResourceCount: named,
		NameCount:          len(file.Names),
		TotalPayloadBytes:  total,
		DecodedCount:       decoded,
	}
}

func buildInfo(file *lvrsrc.File) WASMInfo {
	info := WASMInfo{
		Deps: WASMDeps{
			FrontPanel:   []WASMDepEntry{},
			BlockDiagram: []WASMDepEntry{},
		},
	}
	if file == nil {
		return info
	}

	ctx := codecs.Context{
		FileVersion: file.Header.FormatVersion,
		Kind:        file.Kind,
	}

	// Display name: first LVSR section's Name, fallback to any non-empty
	// section name, otherwise empty.
	info.DisplayName = firstSectionName(file, "LVSR")

	// Version (vers).
	if payload, ok := firstPayload(file, string(vers.FourCC)); ok {
		if raw, err := (vers.Codec{}).Decode(ctx, payload); err == nil {
			if v, ok := raw.(vers.Value); ok {
				info.Version = v.Text
			}
		}
	}

	// Description (STRG).
	if payload, ok := firstPayload(file, string(strg.FourCC)); ok {
		if raw, err := (strg.Codec{}).Decode(ctx, payload); err == nil {
			if s, ok := raw.(strg.Value); ok && s.Text != "" {
				info.Description = s.Text
				info.HasDesc = true
			}
		}
	}

	// Icon — prefer colour (icl8 > icl4 > ICON). The picker falls through
	// each variant that is missing, mis-sized, or fails to decode. Pixels
	// are pre-expanded to RGBA so the browser skips the palette lookup.
	if pick, ok := iconcodec.PickBest(ctx, func(fourCC string) ([]byte, bool) {
		return firstPayload(file, fourCC)
	}); ok {
		info.Icon = &WASMIcon{
			FourCC: pick.FourCC,
			Width:  pick.Value.Width,
			Height: pick.Value.Height,
			RGBA:   base64.StdEncoding.EncodeToString(pick.Value.RGBA()),
		}
	}

	// Dependencies — LIfp (front panel) and LIbd (block diagram).
	if payload, ok := firstPayload(file, string(lifp.FourCC)); ok {
		info.Deps.FrontPanel = decodeLIfp(ctx, payload)
	}
	if payload, ok := firstPayload(file, string(libd.FourCC)); ok {
		info.Deps.BlockDiagram = decodeLIbd(ctx, payload)
	}

	return info
}

func firstPayload(file *lvrsrc.File, fourCC string) ([]byte, bool) {
	for _, block := range file.Blocks {
		if block.Type != fourCC {
			continue
		}
		if len(block.Sections) == 0 {
			continue
		}
		return block.Sections[0].Payload, true
	}
	return nil, false
}

func firstSectionName(file *lvrsrc.File, fourCC string) string {
	for _, block := range file.Blocks {
		if block.Type != fourCC {
			continue
		}
		for _, section := range block.Sections {
			if section.Name != "" {
				return section.Name
			}
		}
	}
	return ""
}

func decodeLIfp(ctx codecs.Context, payload []byte) []WASMDepEntry {
	raw, err := (lifp.Codec{}).Decode(ctx, payload)
	if err != nil {
		return []WASMDepEntry{}
	}
	v, ok := raw.(lifp.Value)
	if !ok {
		return []WASMDepEntry{}
	}
	out := make([]WASMDepEntry, 0, len(v.Entries))
	for _, entry := range v.Entries {
		out = append(out, WASMDepEntry{
			LinkType:   entry.LinkType,
			Qualifiers: append([]string{}, entry.Qualifiers...),
		})
	}
	return out
}

func decodeLIbd(ctx codecs.Context, payload []byte) []WASMDepEntry {
	raw, err := (libd.Codec{}).Decode(ctx, payload)
	if err != nil {
		return []WASMDepEntry{}
	}
	v, ok := raw.(libd.Value)
	if !ok {
		return []WASMDepEntry{}
	}
	out := make([]WASMDepEntry, 0, len(v.Entries))
	for _, entry := range v.Entries {
		out = append(out, WASMDepEntry{
			LinkType:   entry.LinkType,
			Qualifiers: append([]string{}, entry.Qualifiers...),
		})
	}
	return out
}

func successResult(data any) string {
	b, _ := json.Marshal(WASMResult{Success: true, Data: data})
	return string(b)
}

func errorResult(msg string) string {
	b, _ := json.Marshal(WASMResult{Success: false, Error: msg})
	return string(b)
}

func headerToWASM(h lvrsrc.Header) WASMHeader {
	return WASMHeader{
		Magic:         h.Magic,
		FormatVersion: h.FormatVersion,
		Type:          h.Type,
		Creator:       h.Creator,
		InfoOffset:    h.InfoOffset,
		InfoSize:      h.InfoSize,
		DataOffset:    h.DataOffset,
		DataSize:      h.DataSize,
	}
}

func kindName(k lvrsrc.FileKind) string {
	switch k {
	case lvrsrc.FileKindVI:
		return "VI"
	case lvrsrc.FileKindControl:
		return "Control"
	case lvrsrc.FileKindTemplate:
		return "Template"
	case lvrsrc.FileKindLibrary:
		return "Library"
	default:
		return "Unknown"
	}
}
