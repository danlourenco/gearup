import { describe, it, expect } from "bun:test"
import { checkBrewCask, installBrewCask } from "./brew-cask"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { BrewCaskStep } from "../schema"

describe("checkBrewCask", () => {
  it("runs the default `brew list --cask` check", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "iTerm2", cask: "iterm2" }
    const result = await checkBrewCask(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["brew", "list", "--cask", "iterm2"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns installed=false on non-zero exit", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "iTerm2", cask: "iterm2" }
    const result = await checkBrewCask(step, ctx)

    expect(result.installed).toBe(false)
  })

  it("runs step.check in shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = {
      type: "brew-cask",
      name: "iTerm2",
      cask: "iterm2",
      check: 'test -d "/Applications/iTerm.app"',
    }
    const result = await checkBrewCask(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(['test -d "/Applications/iTerm.app"'])
    expect(exec.calls[0]?.shell).toBe(true)
    expect(result.installed).toBe(true)
  })
})

describe("installBrewCask", () => {
  it("runs `brew install --cask <cask>` and returns ok on exit 0", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "iTerm2", cask: "iterm2" }
    const result = await installBrewCask(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["brew", "install", "--cask", "iterm2"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns ok=false with error when exit code is non-zero", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "Cask not found" })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "missing", cask: "missing" }
    const result = await installBrewCask(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("brew install --cask missing")
      expect(result.error).toContain("exit 1")
      expect(result.error).toContain("Cask not found")
    }
  })
})
