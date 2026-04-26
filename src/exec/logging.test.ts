import { describe, it, expect } from "bun:test"
import { LoggingExec } from "./logging"
import { FakeExec } from "./fake"
import { FakeLogger } from "../log/fake"

describe("LoggingExec", () => {
  it("logs argv, exit code, duration, stdout, stderr around each call", async () => {
    const inner = new FakeExec()
    inner.queueResponse({
      exitCode: 0,
      stdout: "hello\nworld",
      stderr: "warning",
      durationMs: 42,
    })

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    const result = await exec.run({ argv: ["echo", "hi"] })

    expect(result.exitCode).toBe(0)
    expect(inner.calls).toHaveLength(1)

    const joined = log.lines.join("\n")
    expect(joined).toContain("> echo hi")
    expect(joined).toContain("(exit 0")
    expect(joined).toContain("hello")
    expect(joined).toContain("world")
    expect(joined).toContain("warning")
  })

  it("does not log stdout/stderr blocks when they are empty", async () => {
    const inner = new FakeExec()
    inner.queueResponse({ exitCode: 0, stdout: "", stderr: "", durationMs: 5 })

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    await exec.run({ argv: ["true"] })

    const joined = log.lines.join("\n")
    expect(joined).toContain("> true")
    expect(joined).toContain("(exit 0, 5ms)")
    expect(joined).not.toContain("stdout:")
    expect(joined).not.toContain("stderr:")
  })

  it("logs even when the inner exec throws (and rethrows)", async () => {
    const inner = new FakeExec()  // empty queue → next run() throws

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    await expect(exec.run({ argv: ["whatever"] })).rejects.toThrow(/unexpected call/)
    const joined = log.lines.join("\n")
    expect(joined).toContain("> whatever")
    expect(joined).toContain("threw:")
  })

  it("annotates timed-out runs", async () => {
    const inner = new FakeExec()
    inner.queueResponse({ exitCode: 124, stdout: "", stderr: "killed", timedOut: true, durationMs: 60_000 })

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    await exec.run({ argv: ["sleep", "infinity"], timeout: 60_000 })
    const joined = log.lines.join("\n")
    expect(joined).toContain("(timed out)")
  })
})
