package harnesskind

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/opencharly/sdk/candywalk"
	"github.com/opencharly/spec/spec"
)

// content_skill.go — kind:skill's OWNED CONTENT (ops.OpContent; DESIGN §1.1 M1, Content -> IR;
// opencharly/sdk#598).
//
// A `skill:` node names the candy that owns it, and its body is derived from its name at the root of
// the repository that authors it (`.agents/skills/<name>/SKILL.md`, plus `references/<stem>.md`; a
// `type: agent` node's `.agents/agents/<name>.md`) — never authored in the node (DESIGN: "a body
// authored inside a node" has no alternative form). This kind is the one place that knows what such a
// node contributes, so the build kit only asks it (sdk deploykit owned_content.go):
//
//   - for ONE BOX (req.Box set) — the skills whose owner is one of the box's composed candies: their
//     files, installed verbatim under the image user's home at the path they have under their repo
//     root, and the ai.opencharly.skill label carrying them whole (routing metadata from the node,
//     prose from the file). The install set and the label set are one walk, so an image can never
//     carry a skill in its label that it does not install, or the reverse.
//   - in PROJECT scope (req.Box empty) — every node, diagnosed whether or not any box composes it:
//     `charly box validate` reports these, which is what catches a body moved where nothing reads it
//     in a repo that composes no box at all (opencharly/opencharly#530).
//
// A box's diagnostics fail its build; the project's fail validate. Every node is visited in sorted
// order, so a gate that reports the first problem names the same node every run (opencharly/sdk#434).
//
// Only the DEFAULT skill tree (candywalk.AgentsSkillsDir) is read: the request does not yet carry a
// candy's declared `skills:` trees, so a body that lives only in one reports "no body" rather than
// shipping without it — opencharly/plugin-harness-kind#14.

// imageOwnerFix is the remedy the operator's ruling names, quoted verbatim in the diagnostic so the
// author reads the action and not only the prohibition (opencharly/sdk#434).
const imageOwnerFix = "move this skill to a package candy that the image composes, and set `owner:` to that candy"

// skillContent answers ops.OpContent for kind:skill.
func skillContent(req spec.OwnedContentRequest) spec.OwnedContentReply {
	var reply spec.OwnedContentReply
	candies := make(map[string]string, len(req.Candies))
	for _, c := range req.Candies {
		candies[c.Name] = c.RepoRoot
	}
	images := make(map[string]bool, len(req.Images))
	for _, n := range req.Images {
		images[n] = true
	}
	projectScope := req.Box == ""

	nodes := make([]string, 0, len(req.Nodes))
	for node := range req.Nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	seenLabel := map[string]string{} // label family/name -> the node that claimed it
	seenPath := map[string]string{}  // install path -> the node that claimed it
	var entries []spec.LabelSkillEntry
	for _, node := range nodes {
		var s spec.Skill
		if err := json.Unmarshal(req.Nodes[node], &s); err != nil {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf("skill %q: decode: %v", node, err))
			continue
		}
		name := s.Name
		if name == "" {
			name = node // the node name IS the id when the node declares none (spec#247)
		}
		root, composed := candies[s.Owner]
		if !composed {
			// Not one of this scope's candies. In a box that simply means the box does not compose
			// it; in the project it is a finding — the more specific one when the owner is an image.
			// A name that is BOTH an image and a candy is a candy here: the layer resolves the body.
			if projectScope {
				reply.Diagnostics = append(reply.Diagnostics, ownerDiagnostic(name, s, images[s.Owner]))
			}
			continue
		}
		if root == "" {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf(
				"skill %q: the repo root of the owning candy %q cannot be derived from its source dir, so its %s cannot be located",
				name, s.Owner, candywalk.NodeBodyLocation(s.Type, name, nil)))
			continue
		}
		body, err := candywalk.ReadNodeBody(root, name, s.Type, nil)
		if err != nil {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf("skill %q: %v", name, err))
			continue
		}
		if body == nil {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf(
				"skill %q: no body — no %s exists at the repo root of the owning candy %q (a body is derived from its node's name and never authored in the node)",
				name, candywalk.NodeBodyLocation(s.Type, name, nil), s.Owner))
			continue
		}
		if projectScope {
			continue // validation only: the body resolves, which is all the project is asked
		}
		// The install set IS the label set, so a node joins both or neither. Two collisions break
		// that, and each is an authoring error, never a silent collapse:
		//   - an install PATH another node already claimed (the path is keyed by the skill's name
		//     and type, not its family — two families can name the same skill), which would copy
		//     two bodies onto one file while the label listed two entries;
		//   - a label family/name another node already claimed.
		entry := labelEntry(s, name, body)
		key := entry.Family + "/" + entry.Name
		if first, dup := seenLabel[key]; dup {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf(
				"skill %q: node %q has the same family/name %q as node %q — one image can carry one skill per name", name, node, key, first))
			continue
		}
		files, err := skillFiles(root, name, s.Type)
		if err != nil {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf("skill %q: %v", name, err))
			continue
		}
		if path, first, dup := claimedPath(seenPath, files); dup {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf(
				"skill %q: node %q installs %q, which node %q already installs — one image can carry one skill per name", name, node, path, first))
			continue
		}
		seenLabel[key] = node
		for _, f := range files {
			seenPath[f.Path] = node
		}
		reply.Files = append(reply.Files, files...)
		entries = append(entries, entry)
	}
	if len(entries) > 0 {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Family+"/"+entries[i].Name < entries[j].Family+"/"+entries[j].Name
		})
		label, err := json.Marshal(entries)
		if err != nil {
			reply.Diagnostics = append(reply.Diagnostics, fmt.Sprintf("skill label: %v", err))
			return reply
		}
		reply.Labels = map[string]spec.RawBody{spec.LabelSkill: label}
	}
	return reply
}

// ownerDiagnostic names why a node's owner is not a candy of the project. An image owner cannot work,
// and the reason is structural rather than a missing search path: the loader folds a `base:`/`from:`
// node into the BOX set and never the layer set, and only a candy carries a source directory — so no
// placement of the body can make it reachable, and the node would be silently absent from every image.
// The rule fires on the SHAPE, independent of body presence.
func ownerDiagnostic(name string, s spec.Skill, isImage bool) string {
	if isImage {
		return fmt.Sprintf("skill %q: owner %q is an image, not a candy — a `candy:` node carrying `base:` or `from:` folds into the image set, so it has no source directory and this skill's prose can never be read (it is also silently absent from the ai.opencharly.skill label): %s (opencharly/sdk#434)",
			name, s.Owner, imageOwnerFix)
	}
	return fmt.Sprintf("skill %q: owner %q is not a candy of this project, so its prose cannot be located: a `candy:` node carrying `base:` or `from:` is an IMAGE (routed to the box set), and only a candy has a source directory to read %s from",
		name, s.Owner, candywalk.NodeBodyLocation(s.Type, name, nil))
}

// claimedPath reports the first of files whose install path a node already claimed.
func claimedPath(seen map[string]string, files []spec.ContentFile) (path, first string, dup bool) {
	for _, f := range files {
		if n, ok := seen[f.Path]; ok {
			return f.Path, n, true
		}
	}
	return "", "", false
}

// skillFiles lists the files of one node's authored body, each at the path a harness inside the image
// reads it from — the SAME path, relative to the home directory, the file has relative to its repo root:
//
//	skill  <repo>/.agents/skills/<name>/SKILL.md (+ references/<stem>.md)  ->  ~/.agents/skills/<name>/...
//	agent  <repo>/.agents/agents/<name>.md                                  ->  ~/.agents/agents/<name>.md
func skillFiles(root, name, nodeType string) ([]spec.ContentFile, error) {
	if candywalk.ShapeOf(nodeType) == candywalk.ShapeAgentFile {
		rel := filepath.Join(candywalk.AgentsAgentsDir, name+".md")
		return []spec.ContentFile{{Source: filepath.Join(root, rel), Path: filepath.ToSlash(rel)}}, nil
	}
	dir := filepath.Join(candywalk.AgentsSkillsDir, name)
	out := []spec.ContentFile{{
		Source: filepath.Join(root, dir, "SKILL.md"),
		Path:   filepath.ToSlash(filepath.Join(dir, "SKILL.md")),
	}}
	refs, err := os.ReadDir(filepath.Join(root, dir, "references"))
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil // most skills carry no split files
		}
		return nil, err
	}
	for _, e := range refs {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		rel := filepath.Join(dir, "references", e.Name())
		out = append(out, spec.ContentFile{Source: filepath.Join(root, rel), Path: filepath.ToSlash(rel)})
	}
	sort.Slice(out[1:], func(i, j int) bool { return out[1+i].Path < out[1+j].Path })
	return out, nil
}

// labelEntry maps one skill onto its ai.opencharly.skill entry: the ROUTING metadata from the node,
// the prose from the file. A node's own `references:` win over the file's split files, as before.
func labelEntry(s spec.Skill, name string, body *candywalk.AuthoredBody) spec.LabelSkillEntry {
	entry := spec.LabelSkillEntry{
		Family:      s.Family,
		Name:        name,
		Owner:       s.Owner,
		Triggers:    s.Triggers,
		Category:    s.Category,
		Description: body.Description,
		Content:     body.Content,
	}
	for _, r := range s.References {
		entry.References = append(entry.References, spec.LabelSkillReference(r)) // S1016 — convertible field-identical types
	}
	if len(entry.References) == 0 {
		for _, r := range body.References {
			entry.References = append(entry.References, spec.LabelSkillReference{Name: r.Stem, Content: r.Content})
		}
	}
	return entry
}
