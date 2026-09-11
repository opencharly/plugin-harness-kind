// schema/docs.cue — the SELF-CONTAINED CUE schema validating the 'docs' KIND's authored
// VALUE (the ONE docs-site generation CONFIG entity per repo: everything 'charly docs
// generate' needs, declared in the repo's charly.yml). Self-contained per the
// skill/hook/marketplace reproduction contract — the same authored wire keys as the core
// #DocsConfig (spec/schema/docs.cue, the single source that generates spec.DocsConfig); the
// host validates against #DocsInput at load (validateAuthoredPluginInput) and this plugin's
// Invoke canonicalises the body back through the core spec.DocsConfig type. Classification
// mirrors skill/hook/marketplace EXACTLY (R3 — no special-casing for docs anywhere).
#DocsInput: close({
	sources?:     #DocsSourcesInput
	marketplace?: #DocsMarketplaceInput
	projections?: #DocsProjectionsInput
	output?:      #DocsOutputInput
	landing?:     #DocsLandingInput
	gates?:       #DocsGatesInput
})

// #DocsSourcesInput — where the reference documentation is GENERATED from.
#DocsSourcesInput: close({
	// compiled — the plugins compiled into the charly binary (the reference corpus:
	// the plugin/kind/verb surfaces they provide are the docs' source material).
	compiled?: #DocsCompiledInput
	// release_repos — bare candy repo names (e.g. plugin-review, plugin-pipeline)
	// resolved like a go.mod require: entry — from the compiled corpus' go.mod when
	// present, else the repo's latest CalVer tag at generation time.
	release_repos?: [...(string & !="")]
	// extra_repos — additional bare repo names fetched + documented OUTSIDE the
	// compiled corpus (e.g. plugin-gh), same go.mod-require-else-tag resolution.
	extra_repos?: [...(string & !="")]
})
#DocsCompiledInput: close({
	enabled?: *true | bool
	// compiled_plugins_path — the charly.yml whose compiled_plugins: list + providers:
	// manifest declare the binary's in-proc corpus.
	compiled_plugins_path?: *"charly/charly.yml" | string & !=""
	// go_mod_path — the go.mod require: source for the release_repos resolution.
	go_mod_path?: *"charly/charly/go.mod" | string & !=""
})

// #DocsMarketplaceInput — the marketplace repo layout (the 'marketplace generate' input).
#DocsMarketplaceInput: close({
	// path — the marketplace repo's root directory (its candy/ + box/ walks source
	// the skill:/hook:/marketplace: corpus the reference pages link to).
	path?: *"marketplace" | string & !=""
})

// #DocsProjectionsInput — the reference projections the generator emits (all on by
// default; a projection's page set is generated only when its toggle is set).
#DocsProjectionsInput: close({
	recipes?:   *true | bool
	cli?:       *true | bool
	providers?: *true | bool
	candy?:     *true | bool
	box?:       *true | bool
	plugin?:    *true | bool
	landing?:   *true | bool
})

// #DocsOutputInput — the hand-authored doc tree the generated pages are spliced into.
#DocsOutputInput: close({
	// hand_authored — the hand-written docs roots (start/concepts/guides/…); the
	// generator merges the generated reference pages beneath them and leaves the
	// trees it does not own untouched.
	hand_authored?: [...(string & !="")]
})

// #DocsLandingInput — the landing page source.
#DocsLandingInput: close({
	// readme — the repo-root README that anchors the landing page.
	readme?: *"README.md" | string & !=""
})

// #DocsGatesInput — the post-generation shape gates (each runs on every generation and
// FAILS the generator when its condition no longer holds).
#DocsGatesInput: close({
	// site_links — every generated page link resolves in the built site.
	site_links?:    *true | bool
	// sidebar_links — every sidebar entry resolves on its page.
	sidebar_links?: *true | bool
	// prune — stale generated pages (projections without a toggle) are removed.
	prune?:         *true | bool
})
