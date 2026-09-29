# plugin-harness-kind

The `skill` / `hook` / `marketplace` / `docs` **harness-surface kinds** — the
first-class entities of the plugins→candies migration.

A `skill:` node is a sibling top-level entity in a candy's `charly.yml` carrying
the FULL inline skill definition (metadata + markdown content); a `hook:` node
carries a `.claude/hooks/*` gate script; a `marketplace:` node carries the single
harness/marketplace config (families + settings); a `docs:` node carries the
docs-site generation config.

## What it provides

| Capability | Surface |
|---|---|
| `kind:skill` | the `skill:` entity — a full inline skill (metadata + content) |
| `kind:hook` | the `hook:` entity — a `.claude/hooks/*` gate script |
| `kind:marketplace` | the `marketplace:` entity — the harness/marketplace config |
| `kind:docs` | the `docs:` entity — the docs-site generation config |

All four are **flat kinds** (`Structural:false` — no deploy members): their
self-contained values ride `op.Params`, are validated at load against this
plugin's served `#SkillInput` / `#HookInput` / `#MarketplaceInput` / `#DocsInput`
schemas, and fold into `uf.PluginKinds["skill"|"hook"|"marketplace"|"docs"][name]`
as opaque JSON.

The plugin is **compiled-in** — the words must be recognized on every parse
(validate/build/deploy) without an out-of-process build. The `skill` bodies feed
both the `ai.opencharly.skills` OCI label and the regenerated marketplace corpus;
`hook` bodies feed the `.claude/hooks/*` emission; the `marketplace` body feeds
the plugin.json / marketplace.json / profiles.json / settings.json surface.

## How to use it

Author the kind in a candy's `charly.yml`:

```yaml
my-skill:
  skill:
    name: my-skill
    family: myfamily
    owner: my-plugin
    description: |-
      What the skill covers and when to load it.
    content: |
      # my-skill
      ...
```

## Layout

- `candy/plugin-harness-kind/` — the plugin module: `plugin.go` (the provider +
  `NewProvider()`/`NewMeta()` + the `OpLoad` decode), `schema/skill.cue`,
  `schema/hook.cue`, `schema/marketplace.cue`, `schema/docs.cue`,
  `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-internals:plugin` — the plugin/provider model, including
  the flat `kind` class. This candy carries no `skill:` entity of its own; the gap
  is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:skills` — skill authoring and the marketplace corpus.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
