import { describe, it, expect, afterEach } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"
import { extractConfigs } from "./extract"

const tmpDir = path.join("/tmp", `gearup-extract-test-${process.pid}`)

afterEach(async () => {
  await fs.rm(tmpDir, { recursive: true, force: true })
})

describe("extractConfigs", () => {
  it("writes all embedded configs to a fresh directory", async () => {
    const result = await extractConfigs(tmpDir, false)

    expect(result.written.length).toBeGreaterThan(0)
    expect(result.skipped).toEqual([])

    const files = await fs.readdir(tmpDir)
    expect(files).toContain("base.jsonc")
    expect(files).toContain("backend.jsonc")

    const base = await fs.readFile(path.join(tmpDir, "base.jsonc"), "utf8")
    expect(base).toContain('"name": "base"')
  })

  it("skips existing files when force is false", async () => {
    await fs.mkdir(tmpDir, { recursive: true })
    await fs.writeFile(path.join(tmpDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const result = await extractConfigs(tmpDir, false)

    expect(result.skipped).toContain("base.jsonc")
    expect(result.written).not.toContain("base.jsonc")

    const base = await fs.readFile(path.join(tmpDir, "base.jsonc"), "utf8")
    expect(base).toContain("my-custom")
  })

  it("overwrites existing files when force is true", async () => {
    await fs.mkdir(tmpDir, { recursive: true })
    await fs.writeFile(path.join(tmpDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const result = await extractConfigs(tmpDir, true)

    expect(result.written).toContain("base.jsonc")
    expect(result.skipped).toEqual([])

    const base = await fs.readFile(path.join(tmpDir, "base.jsonc"), "utf8")
    expect(base).toContain('"name": "base"')
    expect(base).not.toContain("my-custom")
  })

  it("creates the target directory if it doesn't exist", async () => {
    const deepPath = path.join(tmpDir, "deeply", "nested", "dir")
    await extractConfigs(deepPath, false)

    const files = await fs.readdir(deepPath)
    expect(files.length).toBeGreaterThan(0)
  })
})
