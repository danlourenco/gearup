import { describe, it, expect, spyOn, mock, afterEach } from "bun:test"

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
const tmpStateDir = path.join("/tmp", `gearup-run-cmd-test-${process.pid}`)

process.env.XDG_STATE_HOME = tmpStateDir

afterEach(async () => {
  await fs.rm(tmpStateDir, { recursive: true, force: true })
})

describe("run command", () => {
  it("returns 0 on a successful run", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    const tmpMarker = "/tmp/gearup-test-run-success"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__run-success.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "run-success",
        steps: {
          marker: {
            type: "shell",
            install: `touch ${tmpMarker}`,
            check: `test -f ${tmpMarker}`,
          },
        },
      }),
    )

    try {
      const { result } = await runCommand(gearupRunCommand, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(0)
      const output = logSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(output).toContain("run-success")
      expect(output).toContain("marker")
      expect(output).toContain("Log:")
    } finally {
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }
  })

  it("returns 1 on a failed run and prints the log path", async () => {
    const errSpy = spyOn(console, "error").mockImplementation(() => undefined)
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)

    const fixturePath = path.join(fixtures, "__run-fail.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "run-fail",
        steps: {
          "always-fails": {
            type: "shell",
            install: "false",
            check: "false",
          },
        },
      }),
    )

    try {
      const { result } = await runCommand(gearupRunCommand, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(1)
      const errOutput = errSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(errOutput).toContain("Failed at step")
      expect(errOutput).toContain("always-fails")
      expect(errOutput).toContain("Log:")
    } finally {
      errSpy.mockRestore()
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
    }
  })

  it("creates a real log file containing the subprocess output", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    const tmpMarker = "/tmp/gearup-test-real-log"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__run-log-content.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "real-log",
        steps: {
          marker: {
            type: "shell",
            install: `touch ${tmpMarker}`,
            check: `test -f ${tmpMarker}`,
          },
        },
      }),
    )

    try {
      await runCommand(gearupRunCommand, { rawArgs: ["--config", fixturePath] })
    } finally {
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }

    // Verify a log file exists under tmpStateDir/gearup/logs/ and contains the touch invocation.
    const logsDir = path.join(tmpStateDir, "gearup", "logs")
    const entries = await fs.readdir(logsDir)
    expect(entries.length).toBeGreaterThanOrEqual(1)
    const logContent = await fs.readFile(path.join(logsDir, entries[0]!), "utf8")
    expect(logContent).toContain(`touch ${tmpMarker}`)
    expect(logContent).toContain("(exit")
  })
})
