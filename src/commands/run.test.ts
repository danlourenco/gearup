import { describe, it, expect, spyOn, mock } from "bun:test"

// Mock @clack/prompts globally for any elevation flow that runInstall might trigger.
mock.module("@clack/prompts", () => ({
  confirm: mock(async () => true),
  isCancel: () => false,
  intro: mock(),
  outro: mock(),
  note: mock(),
}))

import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { runCommand as gearupRunCommand } from "./run"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("run command", () => {
  it("returns 0 on a successful run", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    const tmpMarker = "/tmp/gearup-test-run-success"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__run-success.jsonc")
    await fs.writeFile(fixturePath, JSON.stringify({
      version: 1,
      name: "run-success",
      steps: {
        marker: {
          type: "shell",
          install: `touch ${tmpMarker}`,
          check: `test -f ${tmpMarker}`,
        },
      },
    }))

    try {
      const { result } = await runCommand(gearupRunCommand, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(0)
      const output = logSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(output).toContain("run-success")
      expect(output).toContain("marker")
    } finally {
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }
  })

  it("returns 1 on a failed run", async () => {
    const errSpy = spyOn(console, "error").mockImplementation(() => undefined)
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)

    const fixturePath = path.join(fixtures, "__run-fail.jsonc")
    await fs.writeFile(fixturePath, JSON.stringify({
      version: 1,
      name: "run-fail",
      steps: {
        "always-fails": {
          type: "shell",
          install: "false",
          check: "false",
        },
      },
    }))

    try {
      const { result } = await runCommand(gearupRunCommand, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(1)
    } finally {
      errSpy.mockRestore()
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
    }
  })
})
