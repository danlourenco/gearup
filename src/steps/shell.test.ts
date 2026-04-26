import { describe, it, expect } from "bun:test"
import { checkShell, installShell } from "./shell"
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

describe("installShell", () => {
  it("runs step.install in shell mode", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "rust",
      install: "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
      check: "command -v rustc",
    }
    const result = await installShell(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual([step.install])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("returns ok=false on non-zero exit", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "boom" })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "broken",
      install: "false",
      check: "command -v broken",
    }
    const result = await installShell(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("shell broken")
      expect(result.error).toContain("exit 1")
    }
  })
})
