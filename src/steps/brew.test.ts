import { describe, it, expect } from "bun:test"
import { checkBrew, installBrew } from "./brew"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { BrewStep } from "../schema"

describe("checkBrew", () => {
  it("runs the default `brew list --formula` check when step.check is absent", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "jq", formula: "jq" }
    const result = await checkBrew(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls).toHaveLength(1)
    expect(exec.calls[0]?.argv).toEqual(["brew", "list", "--formula", "jq"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns installed=false when the default check exits non-zero", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "jq", formula: "jq" }
    const result = await checkBrew(step, ctx)

    expect(result.installed).toBe(false)
  })

  it("runs step.check in a shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewStep = {
      type: "brew",
      name: "git",
      formula: "git",
      check: "command -v git",
    }
    const result = await checkBrew(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["command -v git"])
    expect(exec.calls[0]?.shell).toBe(true)
  })
})

describe("installBrew", () => {
  it("runs `brew install <formula>` and returns ok on exit 0", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "jq", formula: "jq" }
    const result = await installBrew(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["brew", "install", "jq"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns ok=false with error when exit code is non-zero", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "Error: No available formula" })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "nonexistent", formula: "nonexistent" }
    const result = await installBrew(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("brew install nonexistent")
      expect(result.error).toContain("exit 1")
      expect(result.error).toContain("Error: No available formula")
    }
  })
})
