import { describe, it, expect } from "bun:test"
import { checkGitClone, installGitClone } from "./git-clone"
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

  it("expands bare ~ to ctx.env.HOME", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "homedir",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~",
    }
    await checkGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["test", "-d", "/Users/test"])
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

describe("installGitClone", () => {
  it("creates the parent dir, then runs `git clone <repo> <dest>` (no ref)", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p
    exec.queueResponse({ exitCode: 0 })  // git clone
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "/tmp/dotfiles",
    }
    const result = await installGitClone(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls).toHaveLength(2)
    expect(exec.calls[0]?.argv).toEqual(["mkdir", "-p", "/tmp"])
    expect(exec.calls[1]?.argv).toEqual([
      "git", "clone", "git@github.com:me/dotfiles.git", "/tmp/dotfiles",
    ])
  })

  it("includes --branch when step.ref is set", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p
    exec.queueResponse({ exitCode: 0 })  // git clone
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "thing",
      repo: "git@github.com:me/thing.git",
      dest: "/tmp/thing",
      ref: "main",
    }
    await installGitClone(step, ctx)

    expect(exec.calls[1]?.argv).toEqual([
      "git", "clone", "--branch", "main",
      "git@github.com:me/thing.git", "/tmp/thing",
    ])
  })

  it("expands ~/ in dest using ctx.env.HOME", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p
    exec.queueResponse({ exitCode: 0 })  // git clone
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    }
    await installGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["mkdir", "-p", "/Users/test"])
    expect(exec.calls[1]?.argv?.slice(-1)).toEqual(["/Users/test/.dotfiles"])
  })

  it("returns ok=false when git clone fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p ok
    exec.queueResponse({ exitCode: 128, stderr: "fatal: repository not found" })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "missing",
      repo: "git@github.com:me/missing.git",
      dest: "/tmp/missing",
    }
    const result = await installGitClone(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("git clone missing")
      expect(result.error).toContain("exit 128")
    }
  })

  it("returns ok=false when mkdir fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "Permission denied" })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "x",
      repo: "git@github.com:me/x.git",
      dest: "/root/forbidden/x",
    }
    const result = await installGitClone(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("create parent dir")
    }
  })
})
