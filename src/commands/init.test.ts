import { describe, it, expect, spyOn, mock, afterEach } from "bun:test"

mock.module("@clack/prompts", () => ({
  intro: mock(),
  outro: mock(),
  note: mock(),
  cancel: mock(),
  isCancel: () => false,
}))

import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { initCommand } from "./init"

const tmpStateDir = path.join("/tmp", `gearup-init-cmd-test-${process.pid}`)
const originalXdgConfig = process.env.XDG_CONFIG_HOME

afterEach(async () => {
  await fs.rm(tmpStateDir, { recursive: true, force: true })
  if (originalXdgConfig === undefined) {
    delete process.env.XDG_CONFIG_HOME
  } else {
    process.env.XDG_CONFIG_HOME = originalXdgConfig
  }
})

describe("init command", () => {
  it("writes embedded configs to the XDG_CONFIG_HOME path", async () => {
    process.env.XDG_CONFIG_HOME = tmpStateDir
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)

    try {
      const { result } = await runCommand(initCommand, { rawArgs: [] })

      expect(result).toBe(0)

      const targetDir = path.join(tmpStateDir, "gearup", "configs")
      const files = await fs.readdir(targetDir)
      expect(files).toContain("base.jsonc")
      expect(files).toContain("backend.jsonc")
    } finally {
      logSpy.mockRestore()
    }
  })

  it("preserves existing files without --force", async () => {
    process.env.XDG_CONFIG_HOME = tmpStateDir
    const targetDir = path.join(tmpStateDir, "gearup", "configs")
    await fs.mkdir(targetDir, { recursive: true })
    await fs.writeFile(path.join(targetDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    try {
      await runCommand(initCommand, { rawArgs: [] })

      const base = await fs.readFile(path.join(targetDir, "base.jsonc"), "utf8")
      expect(base).toContain("my-custom")
    } finally {
      logSpy.mockRestore()
    }
  })

  it("overwrites existing files with --force", async () => {
    process.env.XDG_CONFIG_HOME = tmpStateDir
    const targetDir = path.join(tmpStateDir, "gearup", "configs")
    await fs.mkdir(targetDir, { recursive: true })
    await fs.writeFile(path.join(targetDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    try {
      await runCommand(initCommand, { rawArgs: ["--force"] })

      const base = await fs.readFile(path.join(targetDir, "base.jsonc"), "utf8")
      expect(base).toContain('"name": "base"')
      expect(base).not.toContain("my-custom")
    } finally {
      logSpy.mockRestore()
    }
  })
})
