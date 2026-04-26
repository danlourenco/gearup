import { describe, it, expect } from "bun:test"
import { FakeLogger } from "./fake"

describe("FakeLogger", () => {
  it("records each line in the order they're written", () => {
    const log = new FakeLogger()
    log.log("first")
    log.log("second")
    log.log("third")

    expect(log.lines).toEqual(["first", "second", "third"])
  })

  it("returns its synthetic path string", () => {
    const log = new FakeLogger("/fake/path.log")
    expect(log.path()).toBe("/fake/path.log")
  })

  it("defaults the synthetic path to <no log>", () => {
    const log = new FakeLogger()
    expect(log.path()).toBe("<no log>")
  })

  it("close() resolves and is idempotent", async () => {
    const log = new FakeLogger()
    await log.close()
    await log.close()
    expect(log.closed).toBe(true)
  })
})
