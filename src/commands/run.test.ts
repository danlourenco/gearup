import { describe, it, expect, spyOn, mock, afterEach } from "bun:test"

const spinnerMock = {
  start: mock<(label: string) => void>(() => undefined),
  message: mock<(label: string) => void>(() => undefined),
  stop: mock<(label: string) => void>(() => undefined),
}

mock.module("@clack/prompts", () => ({
  intro: mock<(label: string) => void>(() => undefined),
  outro: mock<(label: string) => void>(() => undefined),
  cancel: mock<(label: string) => void>(() => undefined),
  note: mock<(message: string, title?: string) => void>(() => undefined),
  spinner: () => spinnerMock,
  confirm: mock(async () => true),
  isCancel: () => false,
}))

import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { runCommand as gearupRunCommand } from "./run"
import * as clack from "@clack/prompts"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")
const tmpStateDir = path.join("/tmp", `gearup-run-cmd-test-${process.pid}`)

process.env.XDG_STATE_HOME = tmpStateDir

afterEach(async () => {
  await fs.rm(tmpStateDir, { recursive: true, force: true })
})

describe("run command (Clack UX)", () => {
  it("emits intro and outro on success; returns 0", async () => {
    const tmpMarker = "/tmp/gearup-test-clack-run-success"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__clack-run-success.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "clack-success",
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
      expect(clack.intro).toHaveBeenCalled()
      expect(clack.outro).toHaveBeenCalled()
      expect(clack.cancel).not.toHaveBeenCalled()
    } finally {
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }
  })

  it("emits cancel + note + outro on failure; returns 1", async () => {
    const fixturePath = path.join(fixtures, "__clack-run-fail.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "clack-fail",
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
      expect(clack.cancel).toHaveBeenCalled()
      expect(clack.outro).toHaveBeenCalled()
    } finally {
      await fs.unlink(fixturePath).catch(() => undefined)
    }
  })
})
