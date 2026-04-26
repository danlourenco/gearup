import { describe, it, expect } from "bun:test"
import { $ } from "bun"
import path from "node:path"
import fs from "node:fs/promises"

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

describe("e2e: gearup run (safe-install)", () => {
  const installMarker = "/tmp/gearup-e2e-marker"
  const postMarker = "/tmp/gearup-e2e-post"
  const fixture = path.join(fixtures, "safe-install.jsonc")
  const cliPath = path.join(repoRoot, "src/cli.ts")

  it("installs a missing step, runs post_install, then is idempotent on re-run", async () => {
    // Arrange: clean state
    await fs.unlink(installMarker).catch(() => undefined)
    await fs.unlink(postMarker).catch(() => undefined)

    try {
      // First run — install path
      const first = await $`bun run ${cliPath} run --config ${fixture}`.quiet().nothrow()
      expect(first.exitCode).toBe(0)
      expect(first.stdout.toString()).toContain("safe-install")
      expect(first.stdout.toString()).toContain("marker")
      expect(first.stdout.toString()).toContain("Done.")

      // Verify side effects
      await fs.access(installMarker)  // throws if missing
      const postContents = await fs.readFile(postMarker, "utf8")
      expect(postContents.trim()).toBe("post-install ran")

      // Second run — should skip (already installed)
      const second = await $`bun run ${cliPath} run --config ${fixture}`.quiet().nothrow()
      expect(second.exitCode).toBe(0)
      expect(second.stdout.toString()).toContain("already installed")
    } finally {
      await fs.unlink(installMarker).catch(() => undefined)
      await fs.unlink(postMarker).catch(() => undefined)
    }
  })

  it("returns non-zero when a step fails", async () => {
    const failFixturePath = path.join(fixtures, "__e2e-fail.jsonc")
    await fs.writeFile(failFixturePath, JSON.stringify({
      version: 1,
      name: "e2e-fail",
      steps: {
        "always-fails": { type: "shell", install: "false", check: "false" },
      },
    }))

    try {
      const result = await $`bun run ${cliPath} run --config ${failFixturePath}`.quiet().nothrow()
      expect(result.exitCode).toBe(1)
      expect(result.stderr.toString() + result.stdout.toString()).toContain("always-fails")
    } finally {
      await fs.unlink(failFixturePath).catch(() => undefined)
    }
  })
})
