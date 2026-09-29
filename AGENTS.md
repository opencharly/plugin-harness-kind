# AGENTS.md — plugin-harness-kind

Standalone plugin repo for the harness-surface kinds (`kind:skill` +
`kind:hook` + `kind:marketplace` + `kind:docs`). The plugin is a Go module at
`candy/plugin-harness-kind/` (module path
`github.com/opencharly/plugin-harness-kind/candy/plugin-harness-kind`); the root
`charly.yml` only declares `discover: candy` so the repo is a project and its
candy is scanned.

Canonical files:

- `candy/plugin-harness-kind/charly.yml` — the `plugin-harness-kind:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-harness-kind/plugin.go` — the provider (`NewProvider()` +
  `NewMeta()`) and the `OpLoad` decode into the core spec types.
- `candy/plugin-harness-kind/schema/skill.cue` / `hook.cue` / `marketplace.cue` /
  `docs.cue` — the self-contained `#*Input` schemas.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model (incl. the flat `kind` class), the per-plugin
  CUE-schema contract, placement. Load before touching the provider or schema.
- `/charly-internals:skills` — skill authoring and the marketplace corpus the
  `skill:` entity feeds.
- `/charly-internals:marketplace` — the corpus generation the `skill:` bodies
  drive.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-harness-kind/` — compile the plugin module.
- `go test ./...` in `candy/plugin-harness-kind/` — the plugin's Go tests.
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.

## Modify this repo

- Edit the `plugin-harness-kind:` candy entity, the Go source, and the four
  `schema/*.cue` files **together** — the schemas are the served declaration
  surface.
- Keep the plugin compiled-in: the four kind words must be recognized on every
  parse without an out-of-process build.
- All four kinds are FLAT (`Structural:false`) — their values ride `op.Params` and
  fold into `uf.PluginKinds` as opaque JSON. Do not add deploy members.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
