//go:build js && wasm

// Package main provides a WebAssembly entry point for inspecting LabVIEW RSRC/VI files in the browser.
package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/CWBudde/lvrsrc/pkg/lvdiff"
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
}

type WASMSummary struct {
	BlockCount         int `json:"block_count"`
	ResourceCount      int `json:"resource_count"`
	NamedResourceCount int `json:"named_resource_count"`
	NameCount          int `json:"name_count"`
	RawTailBytes       int `json:"raw_tail_bytes"`
	TotalPayloadBytes  int `json:"total_payload_bytes"`
	WarningCount       int `json:"warning_count"`
	ErrorCount         int `json:"error_count"`
}

type WASMValidationData struct {
	Summary WASMValidationSummary `json:"summary"`
	Issues  []lvrsrc.Issue        `json:"issues"`
}

type WASMValidationSummary struct {
	IssueCount   int `json:"issue_count"`
	WarningCount int `json:"warning_count"`
	ErrorCount   int `json:"error_count"`
}

type WASMDumpData struct {
	JSON string `json:"json"`
}

type WASMDiffData struct {
	Summary WASMDiffSummary `json:"summary"`
	Diff    *lvdiff.Diff    `json:"diff"`
}

type WASMDiffSummary struct {
	ItemCount      int  `json:"item_count"`
	AddedCount     int  `json:"added_count"`
	RemovedCount   int  `json:"removed_count"`
	ModifiedCount  int  `json:"modified_count"`
	HeaderCount    int  `json:"header_count"`
	BlockCount     int  `json:"block_count"`
	SectionCount   int  `json:"section_count"`
	DecodedCount   int  `json:"decoded_count"`
	HasDifferences bool `json:"has_differences"`
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
	Preview string `json:"preview,omitempty"` // hex preview (first 64 bytes)
}

func main() {
	registerHandler("parseVI", handleParse)
	registerHandler("dumpVI", handleDump)
	registerHandler("validateVI", handleValidate)
	registerHandler("diffVI", handleDiff)
	registerHandler("resourcePayloadVI", handleResourcePayload)
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
	issues := file.Validate()
	resources := buildResources(file)
	warnings, errors := countIssues(issues)

	result := WASMParseData{
		Kind:            kindName(file.Kind),
		Compression:     string(file.Compression),
		Summary:         buildSummary(file, resources, warnings, errors),
		Header:          headerToWASM(file.Header),
		SecondaryHeader: headerToWASM(file.SecondaryHeader),
		Resources:       resources,
	}
	return result, nil
}

func handleDump(args []js.Value) (any, error) {
	file, err := parseFileArg(args, 0)
	if err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode dump: %w", err)
	}
	return WASMDumpData{JSON: string(b)}, nil
}

func handleValidate(args []js.Value) (any, error) {
	file, err := parseFileArg(args, 0)
	if err != nil {
		return nil, err
	}
	issues := file.Validate()
	if issues == nil {
		issues = []lvrsrc.Issue{}
	}
	warnings, errors := countIssues(issues)
	return WASMValidationData{
		Summary: WASMValidationSummary{
			IssueCount:   len(issues),
			WarningCount: warnings,
			ErrorCount:   errors,
		},
		Issues: issues,
	}, nil
}

func handleDiff(args []js.Value) (any, error) {
	left, err := parseFileArg(args, 0)
	if err != nil {
		return nil, err
	}
	right, err := parseFileArg(args, 1)
	if err != nil {
		return nil, err
	}
	diff := lvdiff.Files(left, right)
	return WASMDiffData{
		Summary: buildDiffSummary(diff),
		Diff:    diff,
	}, nil
}

func handleResourcePayload(args []js.Value) (any, error) {
	file, err := parseFileArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(args) < 3 {
		return nil, fmt.Errorf("resource lookup requires file bytes, type, and id")
	}
	resourceType := args[1].String()
	resourceID := int32(args[2].Int())
	for _, block := range file.Blocks {
		if block.Type != resourceType {
			continue
		}
		for _, section := range block.Sections {
			if section.Index != resourceID {
				continue
			}
			return struct {
				Type    string `json:"type"`
				ID      int32  `json:"id"`
				Name    string `json:"name,omitempty"`
				Size    int    `json:"size"`
				Payload string `json:"payload"`
			}{
				Type:    block.Type,
				ID:      section.Index,
				Name:    section.Name,
				Size:    len(section.Payload),
				Payload: hexEncode(section.Payload),
			}, nil
		}
	}
	return nil, fmt.Errorf("resource %s/%d not found", resourceType, resourceID)
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

func buildResources(file *lvrsrc.File) []WASMResource {
	resources := make([]WASMResource, 0, len(file.Resources()))
	for _, block := range file.Blocks {
		for _, section := range block.Sections {
			preview := section.Payload
			if len(preview) > 64 {
				preview = preview[:64]
			}
			resources = append(resources, WASMResource{
				Type:    block.Type,
				ID:      section.Index,
				Name:    section.Name,
				Size:    len(section.Payload),
				Preview: hexEncode(preview),
			})
		}
	}
	return resources
}

func buildSummary(file *lvrsrc.File, resources []WASMResource, warnings, errors int) WASMSummary {
	namedResources := 0
	totalPayloadBytes := 0
	for _, resource := range resources {
		if resource.Name != "" {
			namedResources++
		}
		totalPayloadBytes += resource.Size
	}
	return WASMSummary{
		BlockCount:         len(file.Blocks),
		ResourceCount:      len(resources),
		NamedResourceCount: namedResources,
		NameCount:          len(file.Names),
		RawTailBytes:       len(file.RawTail),
		TotalPayloadBytes:  totalPayloadBytes,
		WarningCount:       warnings,
		ErrorCount:         errors,
	}
}

func countIssues(issues []lvrsrc.Issue) (warnings, errors int) {
	for _, issue := range issues {
		switch issue.Severity {
		case lvrsrc.SeverityWarning:
			warnings++
		case lvrsrc.SeverityError:
			errors++
		}
	}
	return warnings, errors
}

func buildDiffSummary(diff *lvdiff.Diff) WASMDiffSummary {
	if diff == nil {
		return WASMDiffSummary{}
	}
	var summary WASMDiffSummary
	summary.ItemCount = len(diff.Items)
	summary.HasDifferences = !diff.IsEmpty()
	for _, item := range diff.Items {
		switch item.Category {
		case lvdiff.CategoryAdded:
			summary.AddedCount++
		case lvdiff.CategoryRemoved:
			summary.RemovedCount++
		case lvdiff.CategoryModified:
			summary.ModifiedCount++
		}
		switch item.Kind {
		case lvdiff.KindHeader:
			summary.HeaderCount++
		case lvdiff.KindBlock:
			summary.BlockCount++
		case lvdiff.KindSection:
			summary.SectionCount++
		case lvdiff.KindDecoded:
			summary.DecodedCount++
		}
	}
	return summary
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

const hexChars = "0123456789abcdef"

func hexEncode(b []byte) string {
	buf := make([]byte, len(b)*2)
	for i, v := range b {
		buf[i*2] = hexChars[v>>4]
		buf[i*2+1] = hexChars[v&0xf]
	}
	return string(buf)
}
