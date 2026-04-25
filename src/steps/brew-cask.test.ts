import { describe, it, expect } from "bun:test"
import { checkBrewCask } from "./brew-cask"
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
