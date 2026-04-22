package main

import (
	"errors"
	"testing"
)

func TestExitCodeForError(t *testing.T) {
	if got, want := exitCodeForError(nil), 0; got != want {
		t.Fatalf("exitCodeForError(nil) = %d, want %d", got, want)
	}

	if got, want := exitCodeForError(errors.New("boom")), 1; got != want {
		t.Fatalf("exitCodeForError(generic) = %d, want %d", got, want)
	}

	if got, want := exitCodeForError(&exitCodeError{code: 2, err: errors.New("bad")}), 2; got != want {
		t.Fatalf("exitCodeForError(exitCodeError) = %d, want %d", got, want)
	}
}

func asExitCodeError(err error, target **exitCodeError) bool {
	return errors.As(err, target)
}
