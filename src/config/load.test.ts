import { describe, it, expect } from "bun:test"
import { loadConfig } from "./load"
import path from "node:path"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("loadConfig", () => {
  it("loads a JSONC file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.jsonc"))
    expect(config.name).toBe("single-brew")
    expect(config.steps).toHaveLength(1)
    expect(config.steps?.[0]?.type).toBe("brew")
  })

  it("loads a YAML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.yaml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
  })

  it("loads a TOML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.toml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
  })

  it("loads a config with all five step types", async () => {
    const config = await loadConfig(path.join(fixtures, "all-five-types.jsonc"))
    expect(config.steps).toHaveLength(5)
    const types = config.steps?.map((s) => s.type)
    expect(types).toEqual(["brew", "brew-cask", "curl-pipe-sh", "git-clone", "shell"])
  })

  it("throws with a helpful message when the file is not found", async () => {
    await expect(loadConfig("/nope/missing.jsonc")).rejects.toThrow(/missing\.jsonc/)
  })

  it("throws with schema-validation detail when the file is invalid", async () => {
    // Write an invalid fixture inline for this test
    const fs = await import("node:fs/promises")
    const tmpPath = path.join(fixtures, "__invalid-tmp.jsonc")
    await fs.writeFile(tmpPath, JSON.stringify({ version: 2, name: "bad" }))
    try {
      await expect(loadConfig(tmpPath)).rejects.toThrow()
    } finally {
      await fs.unlink(tmpPath).catch(() => undefined)
    }
  })
})
