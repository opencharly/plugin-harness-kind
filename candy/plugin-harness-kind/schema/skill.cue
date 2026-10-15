// schema/skill.cue — the SELF-CONTAINED CUE schema validating the `skill` KIND's authored VALUE.
// Ships over Describe (schema_cue); references NO base def so it compiles standalone
// (BuildCapabilities compiles it alone, failing loudly if broken) AND splices onto the base
// (the base ++ plugin splice detects a def-name collision, not resolves base refs).
//
// It is a FAITHFUL reproduction of the core #Skill (spec/schema/skill.cue — the single source
// that generates spec.Skill via gengotypes), following the group/agent/… externalized-kind
// pattern: the SAME authored WIRE keys, so the host validates a real skill entity against
// #SkillInput (validateAuthoredPluginInput, the load gate) and this plugin's Invoke canonicalises
// the body back through the core spec.Skill type.
//
// THE BODY LIVES IN A FILE (the `.agents/skills` relocation). A skill is authored at
// <tree>/<name>/SKILL.md plus references/<stem>.md, where <tree> is the owning candy's `skills:`
// list (default `.agents/skills`) and <name> is this node's `name:` — or the NODE NAME when
// `name:` is omitted. So `name:`, `description:` and `content:` are OPTIONAL: a node may carry only
// the ROUTING the consumers need (`family`, `owner`, `triggers`, `category`) while the file carries
// the prose. The in-line form stays ACCEPTED until the relocation's final leg; the core schema
// states the same contract, and the two move together (this schema is its faithful reproduction).
#SkillInput: close({
	name?:       string & =~"^[a-z][a-z0-9-]*$" // skill id = the SKILL.md directory name; DEFAULTS to the node name
	family:      string & =~"^[a-z][a-z0-9-]*$" // marketplace family → <family>/skills/
	owner:       string & =~"^[a-z][a-z0-9-]*$" // owning candy/concept-candy entity name
	description?: string & !=""                  // SKILL.md frontmatter description — authored in the FILE once the body moved
	content?:    string & !=""                   // SKILL.md markdown body — authored in the FILE once the body moved
	type?:       *"skill" | "agent"              // "agent" ⇒ a sub-agent definition
	model?:      string & !=""                   // agent type: frontmatter model
	tools?:      [...(string & !="")]            // agent type: allowed tool set
	references?: [...#SkillReferenceInput]       // SKILL.md references/<stem>.md split files
	triggers?:   [...(string & !="")]            // R0-dispatcher trigger phrases
	category?:   *"development" | "commands" | "kind" | "images"
})
#SkillReferenceInput: close({
	name:    string & =~"^[a-z0-9][a-z0-9-]*$" // file stem (references/<name>.md)
	content: string & !=""
})
