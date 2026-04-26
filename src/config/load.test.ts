import { describe, it, expect } from "bun:test"
import { loadConfig } from "./load"
import path from "node:path"
import fs from "node:fs/promises"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("loadConfig", () => {
  it("loads a JSONC file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.jsonc"))
    expect(config.name).toBe("single-brew")
    expect(config.steps).toHaveLength(1)
    expect(config.steps?.[0]?.type).toBe("brew")
    expect(config.steps?.[0]?.name).toBe("jq")
  })

  it("loads a YAML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.yaml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
    expect(config.steps?.[0]?.name).toBe("jq")
  })

  it("loads a TOML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.toml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
    expect(config.steps?.[0]?.name).toBe("jq")
  })

  it("loads a config with all five step types", async () => {
    const config = await loadConfig(path.join(fixtures, "all-five-types.jsonc"))
    expect(config.steps).toHaveLength(5)
    const types = config.steps?.map((s) => s.type)
    expect(types).toEqual([
      "brew",
      "brew-cask",
      "curl-pipe-sh",
      "git-clone",
      "shell",
    ])
  })

  it("throws with a helpful message when the file is not found", async () => {
    await expect(loadConfig("/nope/missing.jsonc")).rejects.toThrow(/missing\.jsonc/)
  })

  it("throws with schema-validation detail when the file is invalid", async () => {
    const tmpPath = path.join(fixtures, "__invalid-tmp.jsonc")
    await fs.writeFile(tmpPath, JSON.stringify({ version: 2, name: "bad" }))
    try {
      await expect(loadConfig(tmpPath)).rejects.toThrow()
    } finally {
      await fs.unlink(tmpPath).catch(() => undefined)
    }
  })

  it("resolves extends and merges step records", async () => {
    const config = await loadConfig(path.join(fixtures, "extends-child.jsonc"))
    // Child extends base; base has jq, child has git → merged: both present
    const names = config.steps?.map((s) => s.name).sort()
    expect(names).toEqual(["git", "jq"])
  })

  it("override semantics: child step with same key overrides base step", async () => {
    const config = await loadConfig(path.join(fixtures, "extends-override.jsonc"))
    // Base has jq as { type: "brew", formula: "jq" }
    // Override has jq as { type: "shell", install: "echo override", check: "false" }
    // defu defaults: child wins on key collision
    const jq = config.steps?.find((s) => s.name === "jq")
    expect(jq?.type).toBe("shell")
    if (jq?.type === "shell") {
      expect(jq.install).toBe("echo override")
    }
  })

  it("works with a config path that has no extension", async () => {
    // c12's auto-extension resolution should find single-brew.jsonc when given just "single-brew"
    const config = await loadConfig(path.join(fixtures, "single-brew"))
    expect(config.name).toBe("single-brew")
  })
})
