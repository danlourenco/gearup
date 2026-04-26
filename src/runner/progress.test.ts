import { describe, it, expect } from "bun:test"
import { FakeReporter, NoopReporter } from "./progress"

describe("FakeReporter", () => {
  it("records start/update/finish events in order", () => {
    const r = new FakeReporter()
    r.start("a")
    r.update("a-mid")
    r.finish("a-done")
    r.start("b")
    r.finish("b-done")

    expect(r.events).toEqual([
      { kind: "start", label: "a" },
      { kind: "update", label: "a-mid" },
      { kind: "finish", label: "a-done" },
      { kind: "start", label: "b" },
      { kind: "finish", label: "b-done" },
    ])
  })
})

describe("NoopReporter", () => {
  it("does nothing without throwing", () => {
    const r = new NoopReporter()
    r.start("a")
    r.update("b")
    r.finish("c")
    expect(true).toBe(true)
  })
})
