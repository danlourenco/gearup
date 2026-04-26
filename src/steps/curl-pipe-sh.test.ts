import { describe, it, expect } from "bun:test"
import { checkCurlPipe, installCurlPipe } from "./curl-pipe-sh"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { CurlPipeStep } from "../schema"

describe("checkCurlPipe", () => {
  it("runs step.check in shell mode", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    }
    const result = await checkCurlPipe(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["command -v brew"])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("returns installed=false when the check fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    }
    const result = await checkCurlPipe(step, ctx)

    expect(result.installed).toBe(false)
  })
})

describe("installCurlPipe", () => {
  it("runs `curl -fsSL <url> | bash` by default", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    }
    const result = await installCurlPipe(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["curl -fsSL https://example.com/install.sh | bash"])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("uses step.shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "rust",
      url: "https://sh.rustup.rs",
      shell: "sh",
      check: "command -v rustc",
    }
    await installCurlPipe(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["curl -fsSL https://sh.rustup.rs | sh"])
  })

  it("appends `-s -- <args>` when args is non-empty", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "rust",
      url: "https://sh.rustup.rs",
      shell: "sh",
      args: ["-y", "--default-toolchain", "stable"],
      check: "command -v rustc",
    }
    await installCurlPipe(step, ctx)

    expect(exec.calls[0]?.argv).toEqual([
      "curl -fsSL https://sh.rustup.rs | sh -s -- -y --default-toolchain stable",
    ])
  })

  it("returns ok=false with error on non-zero exit", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 22, stderr: "404 Not Found" })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "missing",
      url: "https://example.com/404.sh",
      check: "command -v missing",
    }
    const result = await installCurlPipe(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("curl-pipe-sh missing")
      expect(result.error).toContain("exit 22")
    }
  })
})
