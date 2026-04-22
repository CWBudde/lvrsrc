package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// The schemas in docs/schemas/ are the public contract for the --json output
// of `dump` and `validate`. The tests here guarantee that:
//
//  1. Each schema file parses as JSON.
//  2. Its declared top-level required keys exactly match the top-level keys
//     the CLI actually emits on representative corpus input.
//  3. $defs named in root.required via $ref are themselves well-formed.
//
// This does not perform full JSON Schema validation (no external dependency),
// but it catches the common drift scenarios: adding/removing a top-level
// field and forgetting to update the schema, or typos in the schema file.

type jsonSchema struct {
	Schema     string                     `json:"$schema"`
	ID         string                     `json:"$id"`
	Title      string                     `json:"title"`
	Type       string                     `json:"type"`
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
	Defs       map[string]json.RawMessage `json:"$defs"`
}

func loadSchema(t *testing.T, name string) jsonSchema {
	t.Helper()
	path := filepath.Join(schemasDir(t), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var s jsonSchema
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
	if s.Schema == "" {
		t.Fatalf("%s: missing $schema", name)
	}
	if s.Title == "" {
		t.Fatalf("%s: missing title", name)
	}
	if s.Type != "object" {
		t.Fatalf("%s: root type = %q, want object", name, s.Type)
	}
	if len(s.Required) == 0 {
		t.Fatalf("%s: root required list is empty", name)
	}
	return s
}

func schemasDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "docs", "schemas")
}

func topLevelKeys(t *testing.T, raw []byte) []string {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("Unmarshal top-level error = %v: %s", err, raw)
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestDumpSchemaMatchesCLIOutput(t *testing.T) {
	schema := loadSchema(t, "dump.schema.json")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"dump", fixturePath(t, "config-data.ctl"), "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := topLevelKeys(t, stdout.Bytes())
	want := append([]string(nil), schema.Required...)
	sort.Strings(want)

	if !stringSlicesEqual(got, want) {
		t.Fatalf("dump JSON keys differ from schema required list\n got  = %v\n want = %v", got, want)
	}
	for _, key := range want {
		if _, ok := schema.Properties[key]; !ok {
			t.Fatalf("dump schema declares required=%q but has no properties entry for it", key)
		}
	}
}

func TestValidateSchemaMatchesCLIOutput(t *testing.T) {
	schema := loadSchema(t, "validate.schema.json")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"validate", fixturePath(t, "config-data.ctl"), "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := topLevelKeys(t, stdout.Bytes())
	want := append([]string(nil), schema.Required...)
	sort.Strings(want)
	if !stringSlicesEqual(got, want) {
		t.Fatalf("validate JSON keys differ from schema required list\n got  = %v\n want = %v", got, want)
	}
}

func TestValidateSchemaIssueShape(t *testing.T) {
	// Produce at least one issue so we can assert the issue shape.
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"validate", writeWarningFixture(t), "--json"})
	_ = cmd.Execute() // exit code 1 is expected; we only care about JSON body

	var payload struct {
		Issues []map[string]json.RawMessage `json:"issues"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v\noutput=%s", err, stdout.String())
	}
	if len(payload.Issues) == 0 {
		t.Fatal("expected at least one issue in output")
	}

	schema := loadSchema(t, "validate.schema.json")
	issueDef := map[string]any{}
	if raw, ok := schema.Defs["issue"]; ok {
		if err := json.Unmarshal(raw, &issueDef); err != nil {
			t.Fatalf("unmarshal $defs/issue: %v", err)
		}
	} else {
		t.Fatal("schema missing $defs/issue")
	}
	requiredIssue, _ := issueDef["required"].([]any)
	if len(requiredIssue) == 0 {
		t.Fatal("schema $defs/issue has no required list")
	}

	first := payload.Issues[0]
	for _, raw := range requiredIssue {
		key, _ := raw.(string)
		if _, ok := first[key]; !ok {
			t.Fatalf("issue missing required key %q; got keys=%v", key, mapKeys(first))
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
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

func mapKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
