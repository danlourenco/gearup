import { describe, it, expect } from "bun:test"
import { checkCurlPipe } from "./curl-pipe-sh"
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
