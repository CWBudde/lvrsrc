// Package corpus resolves paths to shared test fixtures under the
// repository-root testdata/ tree. It is intended for use from _test.go
// files only.
package corpus

import (
	"path/filepath"
	"runtime"
)

// Path returns the absolute path to a fixture under testdata/corpus/.
// Use it from tests in any package so fixtures don't need to be duplicated
// per package.
func Path(name string) string {
	return filepath.Join(Dir(), name)
}

// Dir returns the absolute path to the testdata/corpus/ directory.
func Dir() string {
	return filepath.Join(repoRoot(), "testdata", "corpus")
}

// repoRoot returns the absolute path to the repository root, derived from
// the location of this source file at internal/corpus/corpus.go.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}
