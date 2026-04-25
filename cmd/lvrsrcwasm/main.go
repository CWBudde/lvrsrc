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
	"github.com/CWBudde/lvrsrc/internal/codecs/bdpw"
	iconcodec "github.com/CWBudde/lvrsrc/internal/codecs/icon"
	"github.com/CWBudde/lvrsrc/internal/codecs/libd"
	"github.com/CWBudde/lvrsrc/internal/codecs/lifp"
	"github.com/CWBudde/lvrsrc/internal/codecs/lvsr"
	"github.com/CWBudde/lvrsrc/internal/codecs/pthx"
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
	DisplayName string     `json:"display_name,omitempty"`
	Version     string     `json:"version,omitempty"`
	Description string     `json:"description,omitempty"`
	HasDesc     bool       `json:"has_desc"`
	Icon        *WASMIcon  `json:"icon,omitempty"`
	Deps        WASMDeps   `json:"deps"`
	Flags       *WASMFlags `json:"flags,omitempty"`
}

// WASMFlags surfaces the decoded LVSR flag set (plus password presence
// derived by combining LVSR.Locked with BDPW). Only flags whose bit is
// actually set are reported as true; the JS layer renders one chip per
// true flag.
type WASMFlags struct {
	SuspendOnRun       bool `json:"suspend_on_run"`
	Locked             bool `json:"locked"`
	RunOnOpen          bool `json:"run_on_open"`
	SavedForPrevious   bool `json:"saved_for_previous"`
	SeparateCode       bool `json:"separate_code"`
	ClearIndicators    bool `json:"clear_indicators"`
	AutoErrorHandling  bool `json:"auto_error_handling"`
	HasBreakpoints     bool `json:"has_breakpoints"`
	Debuggable         bool `json:"debuggable"`
	PasswordProtected  bool `json:"password_protected"`
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

// WASMDepEntry is one decoded link-info reference. Path fields are
// populated when the embedded PTH0/PTH1 reference decoded cleanly.
type WASMDepEntry struct {
	LinkType    string    `json:"link_type"`
	Qualifiers  []string  `json:"qualifiers"`
	PrimaryPath *WASMPath `json:"primary_path,omitempty"`
}

// WASMPath is the JSON-friendly projection of a typed path reference.
// Components are rendered as strings (caller-side encoding); the
// classification booleans summarise the path's TPIdent / TPVal.
type WASMPath struct {
	Ident      string   `json:"ident"`
	TPIdent    string   `json:"tpident,omitempty"`
	Components []string `json:"components"`
	IsAbsolute bool     `json:"is_absolute,omitempty"`
	IsRelative bool     `json:"is_relative,omitempty"`
	IsUNC      bool     `json:"is_unc,omitempty"`
	IsNotAPath bool     `json:"is_not_a_path,omitempty"`
	IsPhony    bool     `json:"is_phony,omitempty"`
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
// is internal/codecs/registry.go and pkg/lvvi.newLvviRegistry.
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
	"LVSR": {},
	"MUID": {},
	"FPSE": {},
	"BDSE": {},
	"VPDP": {},
	"DTHP": {},
	"RTSG": {},
	"LIBN": {},
	"HIST": {},
	"BDPW": {},
	"FPEx": {},
	"BDEx": {},
	"FTAB": {},
	"VITS": {},
	"LIvi": {},
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

	// Flags — surface every set LVSR bit. Combine LVSR.Locked with
	// BDPW's empty-password sentinel to derive PasswordProtected:
	// a VI is "password protected" only when the lock bit is set AND
	// BDPW's password hash is not the MD5 of the empty string.
	info.Flags = decodeFlags(ctx, file)

	return info
}

func decodeFlags(ctx codecs.Context, file *lvrsrc.File) *WASMFlags {
	payload, ok := firstPayload(file, string(lvsr.FourCC))
	if !ok {
		return nil
	}
	raw, err := (lvsr.Codec{}).Decode(ctx, payload)
	if err != nil {
		return nil
	}
	lv, ok := raw.(lvsr.Value)
	if !ok {
		return nil
	}
	flags := &WASMFlags{
		SuspendOnRun:      lv.SuspendOnRun(),
		Locked:            lv.Locked(),
		RunOnOpen:         lv.RunOnOpen(),
		SavedForPrevious:  lv.SavedForPrevious(),
		SeparateCode:      lv.SeparateCode(),
		ClearIndicators:   lv.ClearIndicators(),
		AutoErrorHandling: lv.AutoErrorHandling(),
		HasBreakpoints:    lv.HasBreakpoints(),
		Debuggable:        lv.Debuggable(),
	}

	// Derive PasswordProtected: Locked + BDPW non-empty hash.
	if flags.Locked {
		if pwPayload, ok := firstPayload(file, string(bdpw.FourCC)); ok {
			if pwRaw, err := (bdpw.Codec{}).Decode(ctx, pwPayload); err == nil {
				if pw, ok := pwRaw.(bdpw.Value); ok && pw.HasPassword() {
					flags.PasswordProtected = true
				}
			}
		}
	}
	return flags
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
			LinkType:    entry.LinkType,
			Qualifiers:  append([]string{}, entry.Qualifiers...),
			PrimaryPath: pathRefToWASM(entry.PrimaryPath.Raw),
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
			LinkType:    entry.LinkType,
			Qualifiers:  append([]string{}, entry.Qualifiers...),
			PrimaryPath: pathRefToWASM(entry.PrimaryPath.Raw),
		})
	}
	return out
}

// pathRefToWASM tries to decode a raw embedded path reference through
// internal/codecs/pthx and projects the result to a JSON-friendly form.
// Returns nil when the bytes are missing or fail to decode.
func pathRefToWASM(raw []byte) *WASMPath {
	if len(raw) == 0 {
		return nil
	}
	v, _, err := pthx.Decode(raw)
	if err != nil {
		return nil
	}
	components := make([]string, len(v.Components))
	for i, c := range v.Components {
		components[i] = string(c)
	}
	return &WASMPath{
		Ident:      v.Ident,
		TPIdent:    v.TPIdent,
		Components: components,
		IsAbsolute: v.IsAbsolute(),
		IsRelative: v.IsRelative(),
		IsUNC:      v.IsUNC(),
		IsNotAPath: v.IsNotAPath(),
		IsPhony:    v.IsPhony(),
	}
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
