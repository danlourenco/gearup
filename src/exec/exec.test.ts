import { describe, it, expect } from "bun:test"
import { FakeExec } from "./fake"

describe("FakeExec", () => {
  it("records calls and returns queued responses", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0, stdout: "ok" })

    const result = await exec.run({ argv: ["brew", "list", "--formula", "jq"] })

    expect(result.exitCode).toBe(0)
    expect(result.stdout).toBe("ok")
    expect(exec.calls).toHaveLength(1)
    expect(exec.calls[0]?.argv).toEqual(["brew", "list", "--formula", "jq"])
  })

  it("throws when queue is empty", async () => {
    const exec = new FakeExec()
    await expect(exec.run({ argv: ["brew"] })).rejects.toThrow(/unexpected call/)
  })

  it("fills defaults on queued response", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })

    const result = await exec.run({ argv: ["false"] })

    expect(result.exitCode).toBe(1)
    expect(result.stdout).toBe("")
    expect(result.stderr).toBe("")
    expect(result.timedOut).toBe(false)
  })
})
