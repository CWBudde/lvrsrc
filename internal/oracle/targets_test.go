package oracle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type oracleTargetsDoc struct {
	SchemaVersion int            `json:"schemaVersion"`
	Targets       []oracleTarget `json:"targets"`
}

type oracleTarget struct {
	Name         string   `json:"name"`
	Availability string   `json:"availability"`
	ArtifactRoot string   `json:"artifactRoot"`
	Generator    string   `json:"generator"`
	TestPackage  string   `json:"testPackage"`
	Coverage     []string `json:"coverage"`
	Refresh      string   `json:"refresh"`
	Notes        string   `json:"notes"`
}

func TestOracleTargetsDocumentAvailableHarnesses(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "oracle-targets.json"))
	if err != nil {
		t.Fatalf("ReadFile(docs/oracle-targets.json): %v", err)
	}
	var doc oracleTargetsDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Unmarshal(docs/oracle-targets.json): %v", err)
	}
	if doc.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", doc.SchemaVersion)
	}

	targets := map[string]oracleTarget{}
	for _, target := range doc.Targets {
		if target.Name == "" || target.Availability == "" || target.ArtifactRoot == "" || len(target.Coverage) == 0 || target.Notes == "" {
			t.Fatalf("incomplete oracle target: %+v", target)
		}
		targets[target.Name] = target
	}

	pylabview, ok := targets["pylabview"]
	if !ok {
		t.Fatal("pylabview oracle target missing")
	}
	if pylabview.Availability != "automated" {
		t.Fatalf("pylabview availability = %q, want automated", pylabview.Availability)
	}
	if pylabview.Generator != "scripts/gen-oracle.py" {
		t.Fatalf("pylabview generator = %q, want scripts/gen-oracle.py", pylabview.Generator)
	}
	if pylabview.TestPackage != "./internal/oracle" {
		t.Fatalf("pylabview test package = %q, want ./internal/oracle", pylabview.TestPackage)
	}

	for _, name := range []string{"pylavi", "native-labview"} {
		target, ok := targets[name]
		if !ok {
			t.Fatalf("%s oracle target missing", name)
		}
		if target.Availability == "automated" && (target.Generator == "" || target.TestPackage == "") {
			t.Fatalf("%s is automated but missing generator or test package: %+v", name, target)
		}
	}
}
