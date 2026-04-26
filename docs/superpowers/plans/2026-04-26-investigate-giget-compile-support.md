# Research: Restore `giget` support in the `bun build --compile` binary

> **For the next session loading this brief:** Read this top-to-bottom. It's self-contained — you don't need any context from prior conversations. The goal is to restore `extends: ["github:..."]` and `extends: ["https://..."]` support in the compiled gearup binary without breaking the existing dev-mode flow.
>
> When done, report back with: (a) which approach worked, (b) the actual change in `scripts/build-release.ts` and/or wherever else, (c) a verification command that proves remote extends work in the compiled binary.

## Where the project stands

- **Branch:** `main` (clean, all four port phases merged)
- **Version:** `0.3.0`
- **Stack:** Bun + citty + c12 + confbox + Valibot + execa + @clack/prompts + pathe
- **Tests:** 145 passing across 25 files
- **Compiled binary works** for everything *except* remote `extends:` references

## The problem

`scripts/build-release.ts` currently passes `--external=giget` to `bun build --compile`:

```ts
await $`bun build src/cli.ts --compile --target=${target} --external=giget --outfile=${outfile}`
```

(Look at the script directly to confirm — comment may be inline.)

This was added during Phase 4 because building **without** `--external=giget` fails with an error along these lines:

```
error: No matching export "fetch" in "node-fetch-native/proxy" for import "fetch"
```

(The exact wording may vary across Bun versions.)

**Dependency chain:** `gearup → c12 → giget → node-fetch-native → node-fetch-native/proxy`

The proxy subpath ships a CommonJS shim with `module.exports = { ... }` instead of named ESM exports, and Bun's `--compile` bundler treats this in a way that doesn't surface `fetch` as a named export the way the importer expects.

**Consequence:**
- In dev (`bun run src/cli.ts ...`): giget is loaded normally, remote `extends:` work.
- In the compiled binary: giget is excluded, so any `extends: ["github:..."]` or `extends: ["https://..."]` fails at runtime with a module-not-found error.

The README documents this constraint (see the "Build from source" section), but **we'd like to remove it** because team-shared configs over `github:` is a real, valuable use case.

## Reproduction

To verify the issue exists in the current state:

```bash
cd /Users/dlo/Dev/gearup
export PATH="$HOME/.bun/bin:$PATH"

# 1. Try to build WITHOUT the workaround
bun build src/cli.ts --compile --outfile=/tmp/gearup-without-workaround
# Expected: build fails with the node-fetch-native/proxy error
```

To verify any proposed fix:

```bash
# 2. After applying a fix, build the binary (no --external=giget)
bun build src/cli.ts --compile --outfile=/tmp/gearup-fixed

# 3. Create a temp config that uses a remote extends
mkdir -p /tmp/gearup-test-configs
cat > /tmp/gearup-test-configs/team.jsonc <<EOF
{
  "version": 1,
  "name": "team-test",
  "extends": ["github:danlourenco/gearup/configs/base.jsonc"]
}
EOF

# 4. Run the compiled binary against it (`plan` will try to resolve extends)
/tmp/gearup-fixed plan --config /tmp/gearup-test-configs/team.jsonc
# Expected with a real fix: the 5 steps from base (Homebrew, Git, jq, iTerm2, Bruno) appear in the output.
# Expected without a fix: a module-not-found or ENOENT error, OR the steps from base don't appear because giget didn't run.
```

## Approaches to investigate (in roughly priority order)

### 1. Bundle giget properly with Bun-specific config

Bun's bundler may have flags that handle the CJS shim properly:

- Try `--target=bun` (instead of the default platform target) — bun-target builds may handle CJS differently.
- Check if `bun build` accepts a `--bundler-config` or similar to configure how `node-fetch-native/proxy` is resolved.
- Look for Bun bundler plugins / loaders that convert CJS-with-default-export to ESM-with-named-exports.
- Search GitHub issues on `oven-sh/bun` for "node-fetch-native" or "module.exports fetch" to see if there's a known workaround.

If a bundler-side fix exists, that's the cleanest path — keep using c12's giget integration as-is.

### 2. Patch `node-fetch-native` upstream

Inspect what `node-fetch-native/proxy` actually does. It's likely a fetch polyfill for older Node. If we can:

- Open a PR upstream that adds proper named exports
- Or: use `package.json`'s `"overrides"` (or `"resolutions"` for Yarn) to point `node-fetch-native` to a fork or patched version
- Or: use `bun pm patch` (Bun's patch system) to apply a local fix to the offending file

This is a deeper fix but has the right shape — fix the actual broken thing.

### 3. Replace giget's fetch with something compile-friendly

`giget`'s package.json may let us override its fetch dep. Check if giget exports its dependencies in a swappable way, or if we can fork giget with a slimmer fetch.

### 4. Lazy-load giget at runtime via dynamic import

Currently giget is a static transitive dep — c12 imports it at module load. If c12 (or our wrapper) used `await import("giget")` only when extends actually needs it, the bundler might skip the static analysis path that triggers the CJS issue.

This needs c12 cooperation; unlikely to be fixable from our side without forking c12.

### 5. Custom replacement loader (the fallback)

If none of the above works, the pragmatic fallback: write a tiny custom remote-fetcher (~30 lines) that:

- Detects `github:owner/repo/path` and `https://...` extends entries
- Fetches via Bun's native `fetch` (or `Bun.file` with a URL)
- Writes to a tempfile
- Rewrites the extends entry to the local tempfile path before c12 sees it

This bypasses giget entirely. Loses some giget features (caching across runs, gitlab/bitbucket support, etc.) but covers the 90% use case.

This would live in `src/config/load.ts` as a pre-processing step before c12 is called.

## Files of interest

- `scripts/build-release.ts` — current build pipeline with `--external=giget`
- `src/config/load.ts` — where c12 is invoked; the place to slot in a custom resolver if we go with approach #5
- `package.json` — `"c12": "^2.0.0"`; consider bumping to latest if a newer version handles this differently
- `node_modules/giget/package.json` — inspect to understand its fetch dependency
- `node_modules/node-fetch-native/proxy.js` (or .mjs) — see exactly what the offending shim does
- `README.md` — has a section under "Build from source" documenting the current constraint; update if the fix lands

## Constraints / what NOT to do

- **Don't break dev mode.** `bun run src/cli.ts ...` must continue to work.
- **Don't break the user-facing API.** Configs with `extends: ["github:..."]` should keep working without users changing their configs.
- **Don't break tests.** All 145 tests must still pass after the fix. Run `bun test` to verify.
- **Don't degrade compiled binary size unreasonably.** Current darwin-arm64 binary is ~61MB; if a fix bloats it past ~80MB, weigh the trade-off.

## Success criteria

1. `bun build src/cli.ts --compile --outfile=...` succeeds **without** `--external=giget`.
2. The resulting binary, when run against a config using `extends: ["github:..."]`, successfully resolves the remote config and merges its steps. (See repro steps above.)
3. `bun test` still 145/145 green; `bun run typecheck` exits 0.
4. `scripts/build-release.ts` updated (drop `--external=giget`).
5. `.github/workflows/ci.yml` smoke test still works (it also has `--external=giget` if I recall correctly — verify and remove if the fix is complete).
6. README's "Build from source" note about the giget limitation is removed (or updated if a partial fix lands).

## When you're done

- Make the change(s) in a worktree (don't commit directly to main):
  ```bash
  git worktree add .worktrees/giget-fix -b fix/giget-compile-support
  cd .worktrees/giget-fix
  ```
- Verify, run tests, build the binary, run the repro at the top of this brief.
- Commit and create a PR. Include the verification output in the PR body so a reviewer can see the remote-extends-in-compiled-binary case actually works.
