package harnesskind

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencharly/sdk"
	"github.com/opencharly/sdk/candywalk"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

// content_skill_test.go — kind:skill's owned content (ops.OpContent, opencharly/sdk#598). These are
// the cases that covered sdk's deleted label collector and plugin-box's skill rules, ported to the
// kind that now owns them (D-PLUG-1), plus the install half: the files a composed candy's skills put
// into the image, from the SAME walk as the label.

// writeSkill authors one skill body at <root>/.agents/skills/<name>/SKILL.md with optional references.
func writeSkill(t *testing.T, root, name, frontmatter, body string, refs map[string]string) {
	t.Helper()
	dir := filepath.Join(root, candywalk.AgentsSkillsDir, name)
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\n"+frontmatter+"---\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	for stem, content := range refs {
		if err := os.WriteFile(filepath.Join(dir, "references", stem+".md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// writeAgent authors one agent definition at <root>/.agents/agents/<name>.md.
func writeAgent(t *testing.T, root, name, frontmatter, body string) {
	t.Helper()
	dir := filepath.Join(root, candywalk.AgentsAgentsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte("---\n"+frontmatter+"---\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func node(t *testing.T, s spec.Skill) spec.RawBody {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func boxReq(candies map[string]string, nodes map[string]spec.RawBody) spec.OwnedContentRequest {
	req := spec.OwnedContentRequest{Box: "app", Home: "/home/user", UID: 1000, GID: 1000, Nodes: nodes}
	for name, root := range candies {
		req.Candies = append(req.Candies, spec.OwnedContentCandy{Name: name, RepoRoot: root})
	}
	return req
}

func labelEntries(t *testing.T, reply spec.OwnedContentReply) []spec.LabelSkillEntry {
	t.Helper()
	raw, ok := reply.Labels[spec.LabelSkill]
	if !ok {
		return nil
	}
	var out []spec.LabelSkillEntry
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("the %s label is not a JSON entry array: %v (%s)", spec.LabelSkill, err, raw)
	}
	return out
}

// Only the skills whose owner the box composes ride the label and the install — a skill owned by a
// candy outside the box (or by a concept candy no image composes) contributes nothing.
func TestSkillContent_OnlyTheComposedOwnersSkills(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "postgresql", "name: postgresql\ndescription: pg\n", "\n# pg\n", nil)
	writeSkill(t, repo, "redis", "name: redis\ndescription: redis\n", "\n# redis\n", nil)
	reply := skillContent(boxReq(map[string]string{"postgresql": repo}, map[string]spec.RawBody{
		"postgresql": node(t, spec.Skill{Family: "infra", Owner: "postgresql"}),
		"redis":      node(t, spec.Skill{Family: "infra", Owner: "redis"}),
		"charly":     node(t, spec.Skill{Family: "core", Owner: "charly-core"}),
	}))
	if len(reply.Diagnostics) != 0 {
		t.Fatalf("a box-scope request diagnosed skills the box does not compose: %v", reply.Diagnostics)
	}
	entries := labelEntries(t, reply)
	if len(entries) != 1 || entries[0].Name != "postgresql" {
		t.Fatalf("label = %+v, want only the composed owner's skill", entries)
	}
	if len(reply.Files) != 1 || reply.Files[0].Path != ".agents/skills/postgresql/SKILL.md" {
		t.Fatalf("files = %+v, want only the composed owner's SKILL.md", reply.Files)
	}
}

// The prose comes from the FILE, the routing metadata from the node, and the file's references/ reach
// both the label and the install.
func TestSkillContent_ProseFromTheFileRoutingFromTheNode(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "postgresql", "name: postgresql\ndescription: Use when working with postgresql.\n",
		"\n# Postgresql\nbody from the file\n", map[string]string{"config": "config body from the file\n"})
	reply := skillContent(boxReq(map[string]string{"postgresql": repo}, map[string]spec.RawBody{
		"postgresql": node(t, spec.Skill{Family: "charly-infrastructure", Owner: "postgresql", Triggers: []string{"postgres"}}),
	}))
	e := labelEntries(t, reply)
	if len(e) != 1 {
		t.Fatalf("label = %+v (%v)", e, reply.Diagnostics)
	}
	if e[0].Description != "Use when working with postgresql." || !strings.Contains(e[0].Content, "body from the file") {
		t.Fatalf("prose = %q / %q, want the file's", e[0].Description, e[0].Content)
	}
	if len(e[0].Triggers) != 1 || e[0].Owner != "postgresql" {
		t.Fatalf("routing = %+v, want the node's", e[0])
	}
	if len(e[0].References) != 1 || e[0].References[0].Name != "config" {
		t.Fatalf("references = %+v, want the file's references/config.md", e[0].References)
	}
	want := []string{".agents/skills/postgresql/SKILL.md", ".agents/skills/postgresql/references/config.md"}
	if len(reply.Files) != 2 || reply.Files[0].Path != want[0] || reply.Files[1].Path != want[1] ||
		reply.Files[0].Source != filepath.Join(repo, want[0]) {
		t.Fatalf("files = %+v, want %v sourced at the repo root", reply.Files, want)
	}
}

// A reference-less entry carries no `references` key on the wire, and a file's references DO reach it.
func TestSkillContent_ReferenceLessEntryWireForm(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "postgresql", "name: postgresql\ndescription: d\n", "\n# pg\n", nil)
	reply := skillContent(boxReq(map[string]string{"postgresql": repo}, map[string]spec.RawBody{
		"postgresql": node(t, spec.Skill{Family: "f", Owner: "postgresql"}),
	}))
	if strings.Contains(string(reply.Labels[spec.LabelSkill]), "references") {
		t.Fatalf("a reference-less entry grew a `references` key: %s", reply.Labels[spec.LabelSkill])
	}
}

// The install set IS the label set: one walk produces both.
func TestSkillContent_InstallSetIsTheLabelSet(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "a", "name: a\ndescription: a\n", "\nA\n", nil)
	writeSkill(t, repo, "b", "name: b\ndescription: b\n", "\nB\n", nil)
	writeAgent(t, repo, "v", "name: v\ndescription: v\n", "\nV\n")
	reply := skillContent(boxReq(map[string]string{"own": repo}, map[string]spec.RawBody{
		"a": node(t, spec.Skill{Family: "f", Owner: "own"}),
		"b": node(t, spec.Skill{Family: "f", Owner: "own"}),
		"v": node(t, spec.Skill{Type: "agent", Family: "f", Owner: "own"}),
	}))
	var labelled []string
	for _, e := range labelEntries(t, reply) {
		labelled = append(labelled, e.Name)
	}
	if fmt.Sprint(labelled) != "[a b v]" {
		t.Fatalf("label names = %v, want [a b v]", labelled)
	}
	want := "[.agents/skills/a/SKILL.md .agents/skills/b/SKILL.md .agents/agents/v.md]"
	var paths []string
	for _, f := range reply.Files {
		paths = append(paths, f.Path)
	}
	if fmt.Sprint(paths) != want {
		t.Fatalf("installed = %v, want %s", paths, want)
	}
}

// An agent-type node's body is .agents/agents/<name>.md — never a skills tree — and the shapes do not
// accept each other's location.
func TestSkillContent_AgentBodyComesFromTheAgentsTree(t *testing.T) {
	repo := t.TempDir()
	writeAgent(t, repo, "pr-validator", "name: pr-validator\ndescription: validates\n", "\nagent body\n")
	nodes := map[string]spec.RawBody{"pr-validator": node(t, spec.Skill{Type: "agent", Family: "internals", Owner: "own"})}
	reply := skillContent(boxReq(map[string]string{"own": repo}, nodes))
	e := labelEntries(t, reply)
	if len(e) != 1 || !strings.Contains(e[0].Content, "agent body") || len(reply.Files) != 1 || reply.Files[0].Path != ".agents/agents/pr-validator.md" {
		t.Fatalf("agent: label %+v files %+v diag %v", e, reply.Files, reply.Diagnostics)
	}

	wrong := t.TempDir()
	writeSkill(t, wrong, "pr-validator", "name: pr-validator\ndescription: x\n", "\nin the wrong tree\n", nil)
	reply = skillContent(boxReq(map[string]string{"own": wrong}, nodes))
	if len(reply.Diagnostics) != 1 || !strings.Contains(reply.Diagnostics[0], ".agents/agents/pr-validator.md") || len(reply.Files) != 0 {
		t.Fatalf("an agent body in the SKILLS tree was accepted: files %+v diag %v", reply.Files, reply.Diagnostics)
	}
}

// A composed node with no file is a diagnostic naming the derived location — including one that still
// carries an in-line body, which DESIGN gives no alternative form.
func TestSkillContent_BodylessAndInlineOnlyNodesAreDiagnosed(t *testing.T) {
	repo := t.TempDir()
	reply := skillContent(boxReq(map[string]string{"own": repo}, map[string]spec.RawBody{
		"ghost":  node(t, spec.Skill{Family: "f", Owner: "own"}),
		"inline": node(t, spec.Skill{Family: "f", Owner: "own", Description: "d", Content: "\n# in-line\n"}),
	}))
	if len(reply.Diagnostics) != 2 || !strings.Contains(reply.Diagnostics[0], ".agents/skills/ghost/SKILL.md exists at the repo root of the owning candy") ||
		!strings.Contains(reply.Diagnostics[1], ".agents/skills/inline/SKILL.md") || len(reply.Files) != 0 || reply.Labels != nil {
		t.Fatalf("diag %v files %+v labels %v", reply.Diagnostics, reply.Files, reply.Labels)
	}
}

// A file whose frontmatter names another skill is the wrong prose under the right name: diagnosed.
func TestSkillContent_MalformedFileIsDiagnosed(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "postgresql", "name: mysql\ndescription: d\n", "\nbody\n", nil)
	reply := skillContent(boxReq(map[string]string{"own": repo}, map[string]spec.RawBody{
		"postgresql": node(t, spec.Skill{Family: "f", Owner: "own"}),
	}))
	if len(reply.Diagnostics) != 1 || len(reply.Files) != 0 {
		t.Fatalf("a mismatched file was accepted: diag %v files %+v", reply.Diagnostics, reply.Files)
	}
}

// Project scope (charly box validate): every node is checked whether or not a box composes it, and
// no content is produced — the body at the repo root passes, a body beside the candy is not a home.
func TestSkillContent_ProjectScopeChecksEveryNode(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "review", "name: review\ndescription: r\n", "\nreviews\n", nil)
	nodes := map[string]spec.RawBody{"review": node(t, spec.Skill{Family: "f", Owner: "plugin-review"})}
	ok := skillContent(spec.OwnedContentRequest{Candies: []spec.OwnedContentCandy{{Name: "plugin-review", RepoRoot: repo}}, Nodes: nodes})
	if len(ok.Diagnostics) != 0 || len(ok.Files) != 0 || ok.Labels != nil {
		t.Fatalf("project scope with a resolvable body: %+v", ok)
	}
	beside := t.TempDir()
	writeSkill(t, filepath.Join(beside, "candy", "plugin-review"), "review", "name: review\ndescription: r\n", "\nreviews\n", nil)
	bad := skillContent(spec.OwnedContentRequest{Candies: []spec.OwnedContentCandy{{Name: "plugin-review", RepoRoot: beside}}, Nodes: nodes})
	if len(bad.Diagnostics) != 1 || !strings.Contains(bad.Diagnostics[0], "exists at the repo root of the owning candy") {
		t.Fatalf("a body BESIDE the candy validated clean: %v", bad.Diagnostics)
	}
}

// The owner rules (opencharly/sdk#434): an image owner is rejected with the ruling's message and fix,
// exactly once; an owner that is neither image nor candy is named as unresolvable; a name that is BOTH
// an image and a candy is a candy, and legal.
func TestSkillContent_OwnerRules(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "both", "name: both\ndescription: b\n", "\nb\n", nil)
	reply := skillContent(spec.OwnedContentRequest{
		Candies: []spec.OwnedContentCandy{{Name: "agentteams", RepoRoot: repo}},
		Images:  []string{"pimage", "agentteams"},
		Nodes: map[string]spec.RawBody{
			"by-image":  node(t, spec.Skill{Family: "f", Owner: "pimage", Content: "in-line"}),
			"by-nobody": node(t, spec.Skill{Family: "f", Owner: "ghost-candy"}),
			"both":      node(t, spec.Skill{Family: "f", Owner: "agentteams"}),
		},
	})
	if len(reply.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %v, want exactly one per bad node", reply.Diagnostics)
	}
	if !strings.Contains(reply.Diagnostics[0], `owner "pimage" is an image, not a candy`) || !strings.Contains(reply.Diagnostics[0], imageOwnerFix) {
		t.Fatalf("image owner diagnostic = %q", reply.Diagnostics[0])
	}
	if !strings.Contains(reply.Diagnostics[1], `owner "ghost-candy" is not a candy of this project`) {
		t.Fatalf("unresolvable owner diagnostic = %q", reply.Diagnostics[1])
	}
}

// The order is reproducible: the same tree reports the same diagnostics in the same order every run.
func TestSkillContent_DiagnosticsAreDeterministic(t *testing.T) {
	nodes := map[string]spec.RawBody{}
	for i := 0; i < 8; i++ {
		nodes[fmt.Sprintf("skill-%02d", i)] = node(t, spec.Skill{Family: "f", Owner: fmt.Sprintf("img-%d", i)})
	}
	req := spec.OwnedContentRequest{Nodes: nodes}
	first := fmt.Sprint(skillContent(req).Diagnostics)
	for i := 0; i < 200; i++ {
		if got := fmt.Sprint(skillContent(req).Diagnostics); got != first {
			t.Fatalf("run %d reported %s, run 0 %s", i, got, first)
		}
	}
	if !strings.HasPrefix(first, `[skill "skill-00"`) {
		t.Fatalf("the first diagnostic must be the lowest node name; got %s", first)
	}
}

// The provider speaks the op over the real Invoke envelope, and Describe declares `contents` on
// kind:skill alone — the declaration the host publishes as content_kinds.
func TestInvokeContentAndDescribeContents(t *testing.T) {
	repo := t.TempDir()
	writeSkill(t, repo, "s", "name: s\ndescription: d\n", "\nbody\n", nil)
	params, err := json.Marshal(boxReq(map[string]string{"own": repo}, map[string]spec.RawBody{"s": node(t, spec.Skill{Family: "f", Owner: "own"})}))
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewProvider().Invoke(context.Background(), &pb.InvokeRequest{Op: sdk.OpContent, Reserved: "skill", ParamsJson: params})
	if err != nil {
		t.Fatalf("Invoke(content): %v", err)
	}
	var reply spec.OwnedContentReply
	if err := json.Unmarshal(out.GetResultJson(), &reply); err != nil || len(reply.Files) != 1 {
		t.Fatalf("Invoke(content) reply = %s (%v)", out.GetResultJson(), err)
	}
	if _, err := NewProvider().Invoke(context.Background(), &pb.InvokeRequest{Op: sdk.OpContent, Reserved: "hook", ParamsJson: params}); err == nil {
		t.Fatal("kind:hook answered the content op it does not declare")
	}

	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range caps.GetProvided() {
		if c.GetContents() != (c.GetWord() == "skill") {
			t.Fatalf("kind:%s declares contents=%v", c.GetWord(), c.GetContents())
		}
	}
}

// Two nodes sharing family/name are a diagnostic, never a silent collapse: the label carries ONE
// entry and the install ONE body, so the install set stays the label set (the review note on
// plugin-harness-kind#13: files used to be appended before the dedupe).
func TestSkillContent_DuplicateNameIsDiagnosedNotCollapsed(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeSkill(t, a, "dup", "name: dup\ndescription: a\n", "\nA\n", nil)
	writeSkill(t, b, "dup", "name: dup\ndescription: b\n", "\nB\n", nil)
	reply := skillContent(boxReq(map[string]string{"own-a": a, "own-b": b}, map[string]spec.RawBody{
		"dup":   node(t, spec.Skill{Family: "f", Owner: "own-a"}),
		"dup-2": node(t, spec.Skill{Name: "dup", Family: "f", Owner: "own-b"}),
	}))
	if len(reply.Files) != 1 || len(labelEntries(t, reply)) != 1 {
		t.Fatalf("files %+v, label %+v — want exactly one body and one entry", reply.Files, labelEntries(t, reply))
	}
	if len(reply.Diagnostics) != 1 || !strings.Contains(reply.Diagnostics[0], `same family/name "f/dup"`) {
		t.Fatalf("diagnostics = %v, want the duplicate named", reply.Diagnostics)
	}
}

// The install path is keyed by the skill's name and type, not its family: two families naming the
// same skill would copy two bodies onto ONE file while the label listed two entries. That is a
// diagnostic, so the install set stays the label set (the round-2 review of plugin-harness-kind#13).
func TestSkillContent_SameNameAcrossFamiliesIsDiagnosed(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeSkill(t, a, "dup", "name: dup\ndescription: a\n", "\nA\n", nil)
	writeSkill(t, b, "dup", "name: dup\ndescription: b\n", "\nB\n", nil)
	reply := skillContent(boxReq(map[string]string{"own-a": a, "own-b": b}, map[string]spec.RawBody{
		"dup":   node(t, spec.Skill{Family: "f", Owner: "own-a"}),
		"dup-2": node(t, spec.Skill{Name: "dup", Family: "g", Owner: "own-b"}),
	}))
	if len(reply.Files) != 1 || len(labelEntries(t, reply)) != 1 {
		t.Fatalf("files %+v, label %+v — want exactly one body and one entry", reply.Files, labelEntries(t, reply))
	}
	if len(reply.Diagnostics) != 1 || !strings.Contains(reply.Diagnostics[0], `installs ".agents/skills/dup/SKILL.md", which node "dup" already installs`) {
		t.Fatalf("diagnostics = %v, want the shared install path named", reply.Diagnostics)
	}
}
