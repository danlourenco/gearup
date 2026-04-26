import { describe, it, expect, mock } from "bun:test"

mock.module("@clack/prompts", () => ({
  intro: mock<(label: string) => void>(() => undefined),
  outro: mock<(label: string) => void>(() => undefined),
  note: mock<(message: string, title?: string) => void>(() => undefined),
  cancel: mock<(label: string) => void>(() => undefined),
  spinner: () => ({ start: mock(), message: mock(), stop: mock() }),
  confirm: mock(async () => true),
  isCancel: () => false,
}))

import { runCommand } from "citty"
import path from "node:path"
import { planCommand } from "./plan"
import * as clack from "@clack/prompts"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("plan command (Clack UX)", () => {
  it("returns the runner's exit code (10 when something would install)", async () => {
    const { result } = await runCommand(planCommand, {
      rawArgs: ["--config", path.join(fixtures, "never-installed.jsonc")],
    })
    expect(result).toBe(10)
    expect(clack.intro).toHaveBeenCalled()
    expect(clack.outro).toHaveBeenCalled()
  })

  it("rejects when no config is given (citty raises)", async () => {
    await expect(runCommand(planCommand, { rawArgs: [] })).rejects.toThrow()
  })
})
