package harnesskind

// plugin_test.go — in-tree coverage for the 'docs' ClassKind admission (PR #4:
// the docs-config node admission, spec v0.2026254.503). Each test FAILS without
// the docs word: TestMetaDescribeAdvertisesDocsKind fails when NewMeta drops the
// docs capability, TestInvokeLoadDocsRoundTrip fails when Invoke drops the
// "docs" case, TestInvokeUnknownWordErrors pins the existing unknown-word
// error contract (R10 — the diff must ship coverage that proves the change).
//
// Mirrors the provider tests of the sibling flat kinds: Describe advertises the
// capability + its served InputDef, an OpLoad round-trip decodes the opaque body
// into the core spec.DocsConfig type, unknown words still error.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

// TestMetaDescribeAdvertisesDocsKind — the meta layer MUST advertise the docs
// kind word with its served #DocsInput def (fails when the capability line is
// removed from NewMeta; the schema_cue assertion also fails if schema/docs.cue
// is dropped from the embedded FS).
func TestMetaDescribeAdvertisesDocsKind(t *testing.T) {
	resp, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if resp == nil {
		t.Fatal("Describe: nil reply")
	}
	t.Logf("Describe: calver=%s provided=%d schema_cue=%d bytes",
		resp.GetCalver(), len(resp.GetProvided()), len(resp.GetSchemaCue()))
	for _, c := range resp.GetProvided() {
		t.Logf("Describe: class=%q word=%q input_def=%q", c.GetClass(), c.GetWord(), c.GetInputDef())
	}
	if got := resp.GetSchemaCue(); !strings.Contains(got, "#DocsInput") {
		t.Fatal("Describe: served schema_cue does not define #DocsInput (schema/docs.cue missing from embed?)")
	}
	var docs *pb.ProvidedCapability
	for _, c := range resp.GetProvided() {
		if c.GetClass() == "kind" && c.GetWord() == "docs" {
			docs = c
			break
		}
	}
	if docs == nil {
		t.Fatal(`Describe: kind word "docs" NOT advertised — docs admission missing from NewMeta`)
	}
	if got, want := docs.GetInputDef(), "#DocsInput"; got != want {
		t.Fatalf("Describe: docs InputDef = %q, want %q", got, want)
	}
	t.Logf("Describe: docs capability OK — Class=%q Word=%q InputDef=%q", docs.GetClass(), docs.GetWord(), docs.GetInputDef())
}

// TestInvokeLoadDocsRoundTrip — an OpLoad on the docs word decodes the authored
// body into the core spec.DocsConfig and re-marshals it canonically (fails when
// the "docs" case is removed from Invoke: the load then errors with
// "unsupported word").
func TestInvokeLoadDocsRoundTrip(t *testing.T) {
	// The authored docs: body — the same wire keys as #DocsConfig (schema/docs.cue).
	body := `{"sources":{"compiled":{"enabled":true,"compiled_plugins_path":"charly/charly.yml"},"release_repos":["plugin-gh","plugin-review"]},"output":{"hand_authored":["start","concepts"]},"landing":{"readme":"README.md"},"gates":{"site_links":true,"prune":true}}`
	reply, err := NewProvider().Invoke(context.Background(), &pb.InvokeRequest{
		Op: sdk.OpLoad, Reserved: "docs", Class: "kind", ParamsJson: []byte(body),
	})
	if err != nil {
		t.Fatalf("OpLoad docs: %v — docs word not admitted in Invoke", err)
	}
	if reply == nil {
		t.Fatal("OpLoad docs: nil reply")
	}
	t.Logf("OpLoad: params=%s", body)
	t.Logf("OpLoad: result=%s", string(reply.GetResultJson()))

	var got spec.DocsConfig
	if err := json.Unmarshal(reply.GetResultJson(), &got); err != nil {
		t.Fatalf("OpLoad docs: decode reply into spec.DocsConfig: %v", err)
	}
	if !got.Sources.Compiled.Enabled {
		t.Error("round-trip: sources.compiled.enabled = false, want true")
	}
	if got, want := got.Sources.Compiled.CompiledPluginsPath, "charly/charly.yml"; got != want {
		t.Errorf("round-trip: sources.compiled.compiled_plugins_path = %q, want %q", got, want)
	}
	if len(got.Sources.ReleaseRepos) != 2 || got.Sources.ReleaseRepos[0] != "plugin-gh" || got.Sources.ReleaseRepos[1] != "plugin-review" {
		t.Errorf("round-trip: sources.release_repos = %v, want [plugin-gh plugin-review]", got.Sources.ReleaseRepos)
	}
	if len(got.Output.HandAuthored) != 2 || got.Output.HandAuthored[0] != "start" || got.Output.HandAuthored[1] != "concepts" {
		t.Errorf("round-trip: output.hand_authored = %v, want [start concepts]", got.Output.HandAuthored)
	}
	if got.Landing.Readme != "README.md" {
		t.Errorf("round-trip: landing.readme = %q, want README.md", got.Landing.Readme)
	}
	if !got.Gates.SiteLinks || !got.Gates.Prune || got.Gates.SidebarLinks {
		t.Errorf("round-trip: gates = %+v, want site_links+prune on, sidebar_links off", got.Gates)
	}
}

// TestInvokeUnknownWordErrors — the unknown-word error contract stays intact
// (control: the docs case must not have loosened the default branch).
func TestInvokeUnknownWordErrors(t *testing.T) {
	reply, err := NewProvider().Invoke(context.Background(), &pb.InvokeRequest{
		Op: sdk.OpLoad, Reserved: "nosuchword", ParamsJson: []byte("{}"),
	})
	if err == nil {
		t.Fatal("OpLoad nosuchword: expected error, got nil")
	}
	if reply != nil {
		t.Fatalf("OpLoad nosuchword: expected nil reply, got %v", reply)
	}
	if !strings.Contains(err.Error(), `"nosuchword"`) {
		t.Fatalf("OpLoad nosuchword: error should name the unknown word, got: %v", err)
	}
	t.Logf("OpLoad unknown word: %v", err)
}
