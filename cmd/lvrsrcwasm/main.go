//go:build js && wasm

// Package main provides a WebAssembly entry point for inspecting LabVIEW RSRC/VI files in the browser.
package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/example/lvrsrc/pkg/lvrsrc"
)

// WASMResult is returned as a JSON string to JavaScript.
type WASMResult struct {
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
	Data    *WASMData `json:"data,omitempty"`
}

// WASMData is the web-friendly representation of a parsed RSRC/VI file.
type WASMData struct {
	Kind            string         `json:"kind"`
	Header          WASMHeader     `json:"header"`
	SecondaryHeader WASMHeader     `json:"secondary_header"`
	Resources       []WASMResource `json:"resources"`
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

// WASMResource is a single resource (block section) in the RSRC file.
type WASMResource struct {
	Type    string `json:"type"`
	ID      int32  `json:"id"`
	Name    string `json:"name,omitempty"`
	Size    int    `json:"size"`
	Payload string `json:"payload,omitempty"` // hex preview (first 64 bytes)
}

func main() {
	js.Global().Set("parseVI", js.FuncOf(parseVI))
	select {}
}

func parseVI(_ js.Value, args []js.Value) (out any) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errorResult(fmt.Sprintf("parser panicked: %v", rec))
		}
	}()

	if len(args) < 1 {
		return errorResult("no input data provided")
	}

	jsArray := args[0]
	length := jsArray.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	file, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		return errorResult("failed to parse file: " + err.Error())
	}

	resources := file.Resources()
	wasmResources := make([]WASMResource, 0, len(resources))
	for _, r := range resources {
		wr := WASMResource{
			Type: r.Type,
			ID:   r.ID,
			Name: r.Name,
			Size: r.Size,
		}
		// Attach hex preview from block sections
		for _, block := range file.Blocks {
			if block.Type != r.Type {
				continue
			}
			for _, sec := range block.Sections {
				if sec.Index != r.ID {
					continue
				}
				payload := sec.Payload
				if len(payload) > 64 {
					payload = payload[:64]
				}
				wr.Payload = hexEncode(payload)
			}
		}
		wasmResources = append(wasmResources, wr)
	}

	result := WASMResult{
		Success: true,
		Data: &WASMData{
			Kind:            kindName(file.Kind),
			Header:          headerToWASM(file.Header),
			SecondaryHeader: headerToWASM(file.SecondaryHeader),
			Resources:       wasmResources,
		},
	}
	b, err := json.Marshal(result)
	if err != nil {
		return errorResult("failed to encode result: " + err.Error())
	}
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

const hexChars = "0123456789abcdef"

func hexEncode(b []byte) string {
	buf := make([]byte, len(b)*2)
	for i, v := range b {
		buf[i*2] = hexChars[v>>4]
		buf[i*2+1] = hexChars[v&0xf]
	}
	return string(buf)
}
