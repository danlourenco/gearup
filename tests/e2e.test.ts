import { describe, it, expect } from "bun:test"
import { $ } from "bun"
import path from "node:path"

const repoRoot = path.resolve(import.meta.dir, "..")
const fixtures = path.join(repoRoot, "tests/fixtures")

describe("e2e: gearup plan", () => {
  it("prints a report and exits 10 for a config with un-installed steps", async () => {
    const result = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config ${path.join(fixtures, "never-installed.jsonc")}`.quiet().nothrow()
    const stdout = result.stdout.toString()

    expect(result.exitCode).toBe(10)
    expect(stdout).toContain("never-installed")
    expect(stdout).toContain("always-missing")
    expect(stdout).toContain("would install")
  })

  it("exits non-zero when config is missing", async () => {
    const result = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config /nope/missing.jsonc`.quiet().nothrow()
    expect(result.exitCode).not.toBe(0)
    expect(result.exitCode).not.toBe(10)
  })

  it("accepts YAML and TOML configs", async () => {
    // single-brew.yaml and single-brew.toml both check brew list --formula jq.
    // The exit code depends on host machine state (jq installed or not),
    // but should always be 0 or 10 — never an error code.
    const yamlResult = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config ${path.join(fixtures, "single-brew.yaml")}`.quiet().nothrow()
    const tomlResult = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config ${path.join(fixtures, "single-brew.toml")}`.quiet().nothrow()

    expect([0, 10]).toContain(yamlResult.exitCode)
    expect([0, 10]).toContain(tomlResult.exitCode)
  })
})
