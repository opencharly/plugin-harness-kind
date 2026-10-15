package harnesskind

// skill_input_test.go — coverage for the `skill` kind's FILE-SOURCED shape in the served
// self-contained #SkillInput schema.
//
// WHY THIS TEST EXISTS. The `.agents/skills` relocation moves a skill's body out of its `skill:`
// entity into <tree>/<name>/SKILL.md, leaving the node to carry only the ROUTING metadata
// (family/owner/triggers/category). The core schema (spec/schema/skill.cue) relaxed `name`,
// `description` and `content` to optional. THIS schema is that schema's faithful reproduction and
// the authority the LOAD GATE validates against (validateAuthoredPluginInput), so it has to relax in
// step — otherwise every repository that moves its body would fail its own `charly box validate`
// with `#SkillInput.content: incomplete value`.
//
// Case (a) is the change's reason to exist and is MEASURED against the pre-change schema by the
// leg's gate. Case (b) pins back-compat for the entities that still carry their body in-line, and
// (c)/(d) pin that optionality did not become "anything goes".
//
// It compiles the SAME served schema the load gate compiles, through the SDK's own validator
// (sdk.NewSchemaValidator over the embedded schemaFS), so the test exercises the real gate rather
// than a hand-rolled copy.

import (
	"testing"

	"github.com/opencharly/sdk"
)

func TestSkillInputBodyIsOptional(t *testing.T) {
	v, err := sdk.NewSchemaValidator(schemaFS, "schema")
	if err != nil {
		t.Fatalf("compile the served schema: %v", err)
	}

	// (a) THE NEW BEHAVIOUR — a routing-only node, the shape every moved entity takes, because its
	// markdown now lives at .agents/skills/<name>/SKILL.md.
	routingOnly := []byte(`{
		"family":   "internals",
		"owner":    "charly-internals",
		"triggers": ["editing a skill entity"]
	}`)
	if err := v.ValidateJSON("#SkillInput", routingOnly); err != nil {
		t.Fatalf("#SkillInput rejected a routing-only skill node (the shape a moved entity takes):\n%v", err)
	}
	t.Log("routing-only skill node: ACCEPTED (name/description/content are optional)")

	// (b) BACK-COMPAT — the in-line body every live entity still carries must keep validating, or
	// this change would break the whole corpus on the day it lands.
	inline := []byte(`{
		"name":        "git-workflow",
		"family":      "internals",
		"owner":       "charly-internals",
		"description": "branch-per-change, PR-only landing.",
		"content":     "# git-workflow",
		"triggers":    ["commit / push / open a PR"]
	}`)
	if err := v.ValidateJSON("#SkillInput", inline); err != nil {
		t.Fatalf("#SkillInput rejected an in-line skill body — every live entity still uses this shape:\n%v", err)
	}
	t.Log("in-line skill body: ACCEPTED (back-compat held)")

	// (c) THE ROUTING STAYS REQUIRED — an optional `name`/`description`/`content` must not read as
	// an optional `owner`: `owner` is the OCI-label association (CollectSkills filters by
	// owner ∈ composed candy chain), so an unattributable skill is not a skill. Measured: the
	// served-schema validator reports a missing top-level field as an `incomplete value`, so this
	// arm really does discriminate.
	noOwner := []byte(`{"family": "internals"}`)
	if err := v.ValidateJSON("#SkillInput", noOwner); err == nil {
		t.Fatal("#SkillInput ACCEPTED a skill node with no owner — the OCI-label association would be silently lost")
	}
	t.Log("owner-less skill node: REJECTED (the routing stays required)")

	// (d) AUTHORED-BUT-EMPTY is still a typo, never "no body": the !="" refinement has to survive
	// the relaxation, or a node could publish an empty page and call it a skill.
	emptyBody := []byte(`{"family": "internals", "owner": "charly-internals", "content": ""}`)
	if err := v.ValidateJSON("#SkillInput", emptyBody); err == nil {
		t.Fatal(`#SkillInput ACCEPTED an empty content — the !="" refinement is gone`)
	}
	t.Log(`empty content: REJECTED (the !="" refinement survives)`)

	// (e) The id's shape survives too: `name` is optional, not unbounded.
	badName := []byte(`{"name": "Not-A-Skill-Id", "family": "internals", "owner": "charly-internals"}`)
	if err := v.ValidateJSON("#SkillInput", badName); err == nil {
		t.Fatal("#SkillInput ACCEPTED a malformed name — the skill-id pattern was lost")
	}
	t.Log("malformed name: REJECTED (the id pattern survives)")
}
