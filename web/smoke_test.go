package web_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAppSmoke(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node not installed: %v", err)
	}

	cmd := exec.Command("node", "app_smoke_test.mjs")
	cmd.Dir = webDir(t)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("node app_smoke_test.mjs: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}

func webDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}
