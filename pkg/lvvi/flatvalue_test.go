package lvvi

import (
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// compositeDefault opens a fixture and returns the single composite
// (cluster) front-panel control default it carries. Every composite-default
// corpus fixture has exactly one.
func compositeDefault(t *testing.T, fixture string) TypedConst {
	t.Helper()
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), fixture), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s): %v", fixture, err)
	}
	m, _ := DecodeKnownResources(f)
	defs, ok := m.FrontPanelDefaults()
	if !ok {
		t.Fatalf("%s: no front-panel heap", fixture)
	}
	var composites []TypedConst
	for _, d := range defs {
		if d.CompositeOK {
			composites = append(composites, d)
		}
	}
	if len(composites) != 1 {
		t.Fatalf("%s: got %d composite defaults, want exactly 1", fixture, len(composites))
	}
	return composites[0]
}

// child returns the n-th member of a cluster FlatValue, asserting the kind
// and label match.
func child(t *testing.T, fv *FlatValue, n int, label string, kind FlatKind) FlatValue {
	t.Helper()
	if fv.Kind != FlatKindCluster {
		t.Fatalf("expected cluster, got %s", fv.Kind)
	}
	if n >= len(fv.Children) {
		t.Fatalf("child %d out of range (%d members)", n, len(fv.Children))
	}
	c := fv.Children[n]
	if c.Label != label {
		t.Fatalf("child %d label = %q, want %q", n, c.Label, label)
	}
	if c.Kind != kind {
		t.Fatalf("child %d (%q) kind = %s, want %s", n, label, c.Kind, kind)
	}
	return c
}

// TestFrontPanelDefaultComposite pins the recursive VCTP cluster
// flatten/unflatten decode for every composite (cluster) control default in
// the corpus. The flattened-data layout — members back-to-back in declaration
// order, strings 4-byte-length-prefixed, scalars big-endian, nested clusters
// recursive, a trailing LVVariant kept opaque — is confirmed by each blob
// consuming exactly against its resolved cluster type. resolveCompositeDefault
// recovers that type structurally (the panel DDO points only at LabVIEW's
// internal transaction fields, not the user cluster).
func TestFrontPanelDefaultCompositeDecode(t *testing.T) {
	t.Run("response.ctl", func(t *testing.T) {
		// Response cluster: {id:String, status:String, result:LVVariant}.
		tc := compositeDefault(t, "response.ctl")
		fv := tc.Composite
		if len(fv.Children) != 3 {
			t.Fatalf("got %d members, want 3", len(fv.Children))
		}
		if got := child(t, fv, 0, "id", FlatKindString).String; got != "" {
			t.Fatalf("id = %q, want empty", got)
		}
		if got := child(t, fv, 1, "status", FlatKindString).String; got != "ok" {
			t.Fatalf("status = %q, want %q", got, "ok")
		}
		child(t, fv, 2, "result", FlatKindVariant)
	})

	t.Run("request.ctl", func(t *testing.T) {
		// Request cluster: {id:String, version:I32, command:String, params:LVVariant}.
		tc := compositeDefault(t, "request.ctl")
		fv := tc.Composite
		if len(fv.Children) != 4 {
			t.Fatalf("got %d members, want 4", len(fv.Children))
		}
		if got := child(t, fv, 0, "id", FlatKindString).String; got != "" {
			t.Fatalf("id = %q, want empty", got)
		}
		if got := child(t, fv, 1, "version", FlatKindInt).Int; got != 1 {
			t.Fatalf("version = %d, want 1", got)
		}
		if got := child(t, fv, 2, "command", FlatKindString).String; got != "" {
			t.Fatalf("command = %q, want empty", got)
		}
		child(t, fv, 3, "params", FlatKindVariant)
	})

	t.Run("error-response.ctl", func(t *testing.T) {
		// ErrorResponse cluster: {error:{code:I32, message:String,
		// details:String}, status:String, id:String}.
		tc := compositeDefault(t, "error-response.ctl")
		fv := tc.Composite
		if len(fv.Children) != 3 {
			t.Fatalf("got %d members, want 3", len(fv.Children))
		}
		errc := child(t, fv, 0, "error", FlatKindCluster)
		if got := child(t, &errc, 0, "code", FlatKindInt).Int; got != 0 {
			t.Fatalf("error.code = %d, want 0", got)
		}
		if got := child(t, &errc, 1, "message", FlatKindString).String; got != "" {
			t.Fatalf("error.message = %q, want empty", got)
		}
		if got := child(t, &errc, 2, "details", FlatKindString).String; got != "" {
			t.Fatalf("error.details = %q, want empty", got)
		}
		if got := child(t, fv, 1, "status", FlatKindString).String; got != "error" {
			t.Fatalf("status = %q, want %q", got, "error")
		}
		if got := child(t, fv, 2, "id", FlatKindString).String; got != "" {
			t.Fatalf("id = %q, want empty", got)
		}
	})

	t.Run("ndjson-parser.vi", func(t *testing.T) {
		// ndjson-parser embeds Request.ctl as a TypeDef; the default decodes
		// identically to request.ctl.
		tc := compositeDefault(t, "ndjson-parser.vi")
		fv := tc.Composite
		if len(fv.Children) != 4 {
			t.Fatalf("got %d members, want 4", len(fv.Children))
		}
		if got := child(t, fv, 1, "version", FlatKindInt).Int; got != 1 {
			t.Fatalf("version = %d, want 1", got)
		}
		child(t, fv, 3, "params", FlatKindVariant)
	})

	t.Run("show-panel-argument--cluster.ctl", func(t *testing.T) {
		// Single-member cluster holding a boolean.
		tc := compositeDefault(t, "show-panel-argument--cluster.ctl")
		fv := tc.Composite
		if len(fv.Children) != 1 {
			t.Fatalf("got %d members, want 1", len(fv.Children))
		}
		if got := child(t, fv, 0, "Show Panel?", FlatKindBool).Bool; got != true {
			t.Fatalf("Show Panel? = %v, want true", got)
		}
	})
}

// TestUnflattenValueExplicitType exercises the principled entry point: given
// the governing cluster's flat VCTP index directly (no structural guessing),
// UnflattenValue decodes the same tree, and reports an exact byte fit.
func TestUnflattenValueExplicitType(t *testing.T) {
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), "response.ctl"), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	m, _ := DecodeKnownResources(f)
	tc := compositeDefault(t, "response.ctl")

	fv, n, ok := m.UnflattenValueN(tc.CompositeTypeIndex, tc.Raw)
	if !ok {
		t.Fatal("UnflattenValueN ok = false")
	}
	if n != len(tc.Raw) {
		t.Fatalf("consumed %d bytes, want exact %d", n, len(tc.Raw))
	}
	if fv.Kind != FlatKindCluster || len(fv.Children) != 3 {
		t.Fatalf("decoded %s with %d members, want cluster/3", fv.Kind, len(fv.Children))
	}
	if got := fv.Children[1].String; got != "ok" {
		t.Fatalf("status = %q, want %q", got, "ok")
	}

	// A nonsense index must fail cleanly, not panic or guess.
	if _, _, ok := m.UnflattenValueN(-1, tc.Raw); ok {
		t.Fatal("UnflattenValueN(-1) ok = true, want false")
	}
}
