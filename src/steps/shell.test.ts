import { describe, it, expect } from "bun:test"
import { checkShell } from "./shell"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { ShellStep } from "../schema"

describe("checkShell", () => {
  it("runs step.check in shell mode", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "rust",
      install: "curl ... | sh",
      check: "command -v rustc",
    }
    const result = await checkShell(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["command -v rustc"])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("returns installed=false when the check fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 127 })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "rust",
      install: "curl ... | sh",
      check: "command -v rustc",
    }
    const result = await checkShell(step, ctx)

    expect(result.installed).toBe(false)
  })
})
