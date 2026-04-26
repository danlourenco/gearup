import { describe, it, expect, afterEach, mock } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"

// Mock @clack/prompts BEFORE importing the picker module so the import sees the mock.
const clackSelectMock = mock(async (opts: { options: { value: string }[] }) => opts.options[0]?.value)
const clackIsCancelMock = mock((_v: unknown) => false)
mock.module("@clack/prompts", () => ({
  select: clackSelectMock,
  isCancel: clackIsCancelMock,
  cancel: mock(() => undefined),
}))

import { discoverConfigs } from "./picker"

const userDir = path.join("/tmp", `gearup-discover-user-${process.pid}`)
const projectDir = path.join("/tmp", `gearup-discover-proj-${process.pid}`)

afterEach(async () => {
  await fs.rm(userDir, { recursive: true, force: true })
  await fs.rm(projectDir, { recursive: true, force: true })
  clackSelectMock.mockClear()
  clackIsCancelMock.mockClear()
  clackIsCancelMock.mockImplementation(() => false)
})

async function writeConfig(dir: string, filename: string, name: string, description?: string) {
  await fs.mkdir(dir, { recursive: true })
  const obj: Record<string, unknown> = { version: 1, name }
  if (description) obj.description = description
  await fs.writeFile(path.join(dir, filename), JSON.stringify(obj))
}

describe("discoverConfigs", () => {
  it("returns configs from user dir labeled 'user'", async () => {
    await writeConfig(userDir, "base.jsonc", "base", "Core tools")
    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })
    expect(result).toHaveLength(1)
    expect(result[0]?.name).toBe("base")
    expect(result[0]?.description).toBe("Core tools")
    expect(result[0]?.source).toBe("user")
  })

  it("returns configs from project dir labeled 'project'", async () => {
    await writeConfig(projectDir, "team.jsonc", "Team", "Team setup")
    const result = await discoverConfigs({ userDir: "/nonexistent", projectDir })
    expect(result).toHaveLength(1)
    expect(result[0]?.source).toBe("project")
  })

  it("project-local wins on name collision; user copy is dropped", async () => {
    await writeConfig(userDir, "shared.jsonc", "shared", "User version")
    await writeConfig(projectDir, "shared.jsonc", "shared", "Project version")
    const result = await discoverConfigs({ userDir, projectDir })
    expect(result).toHaveLength(1)
    expect(result[0]?.source).toBe("project")
    expect(result[0]?.description).toBe("Project version")
  })

  it("ignores files without a recognized extension", async () => {
    await fs.mkdir(userDir, { recursive: true })
    await fs.writeFile(path.join(userDir, "README.md"), "not a config")
    await writeConfig(userDir, "real.jsonc", "real")
    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })
    expect(result.map((c) => c.name)).toEqual(["real"])
  })

  it("returns an empty list when both dirs are missing", async () => {
    const result = await discoverConfigs({ userDir: "/nonexistent-a", projectDir: "/nonexistent-b" })
    expect(result).toEqual([])
  })

  it("skips files that fail to parse without crashing the whole discovery", async () => {
    await fs.mkdir(userDir, { recursive: true })
    await fs.writeFile(path.join(userDir, "broken.jsonc"), "{ this is not valid json")
    await writeConfig(userDir, "ok.jsonc", "ok")
    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })
    expect(result.map((c) => c.name)).toEqual(["ok"])
  })
})

import { pickConfig } from "./picker"

describe("pickConfig", () => {
  it("auto-selects when only one config is available (no prompt)", async () => {
    const configs = [
      { name: "only", path: "/tmp/only.jsonc", source: "user" as const },
    ]
    const choice = await pickConfig(configs)
    expect(choice).toBe("/tmp/only.jsonc")
    expect(clackSelectMock).not.toHaveBeenCalled()
  })

  it("prompts via clack.select when multiple configs are available", async () => {
    clackSelectMock.mockImplementation(
      async (opts: { options: { value: string }[] }) => opts.options[1]?.value,
    )

    const configs = [
      { name: "first", path: "/tmp/first.jsonc", source: "user" as const },
      { name: "second", path: "/tmp/second.jsonc", source: "project" as const },
    ]
    const choice = await pickConfig(configs)
    expect(choice).toBe("/tmp/second.jsonc")
    expect(clackSelectMock).toHaveBeenCalled()
  })

  it("returns null when the user cancels", async () => {
    const cancelSym = Symbol.for("clack:cancel")
    clackSelectMock.mockImplementation(async () => cancelSym as unknown as string)
    clackIsCancelMock.mockImplementation(() => true)

    const configs = [
      { name: "a", path: "/tmp/a.jsonc", source: "user" as const },
      { name: "b", path: "/tmp/b.jsonc", source: "user" as const },
    ]
    const choice = await pickConfig(configs)
    expect(choice).toBe(null)
  })

  it("throws when configs list is empty", async () => {
    await expect(pickConfig([])).rejects.toThrow(/no configs/i)
  })
})
