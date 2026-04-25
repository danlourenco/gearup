import { describe, it, expect } from "bun:test"
import { checkGitClone } from "./git-clone"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { GitCloneStep } from "../schema"

describe("checkGitClone", () => {
  it("runs `test -d` on the dest by default", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "/tmp/dotfiles",
    }
    const result = await checkGitClone(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["test", "-d", "/tmp/dotfiles"])
  })

  it("expands ~/ using ctx.env.HOME", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    }
    await checkGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["test", "-d", "/Users/test/.dotfiles"])
  })

  it("returns installed=false when dest does not exist", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "/tmp/missing",
    }
    const result = await checkGitClone(step, ctx)

    expect(result.installed).toBe(false)
  })

  it("runs step.check in shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
      check: "test -f ~/.dotfiles/README.md",
    }
    await checkGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["test -f ~/.dotfiles/README.md"])
    expect(exec.calls[0]?.shell).toBe(true)
  })
})
