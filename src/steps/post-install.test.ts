import { describe, it, expect } from "bun:test"
import { runPostInstall } from "./post-install"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"

describe("runPostInstall", () => {
  it("runs each command in shell mode and returns ok when all succeed", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const result = await runPostInstall(["echo first", "echo second"], "step-name", ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls).toHaveLength(2)
    expect(exec.calls[0]?.argv).toEqual(["echo first"])
    expect(exec.calls[0]?.shell).toBe(true)
    expect(exec.calls[1]?.argv).toEqual(["echo second"])
  })

  it("returns ok on empty list without calling exec", async () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    const result = await runPostInstall([], "step-name", ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls).toHaveLength(0)
  })

  it("stops at the first failing command and reports which one", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    exec.queueResponse({ exitCode: 1, stderr: "boom" })
    const ctx = makeContext({ exec })

    const result = await runPostInstall(
      ["echo ok", "false", "echo never"],
      "colima",
      ctx,
    )

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("colima")
      expect(result.error).toContain("post_install[1]")
      expect(result.error).toContain("exit 1")
      expect(result.error).toContain("boom")
    }
    expect(exec.calls).toHaveLength(2)
  })
})
