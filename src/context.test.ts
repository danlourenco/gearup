import { describe, it, expect } from "bun:test"
import { makeContext } from "./context"
import { FakeExec } from "./exec/fake"

describe("makeContext", () => {
  it("defaults cwd to process.cwd() and env to process.env", () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    expect(ctx.cwd).toBe(process.cwd())
    expect(ctx.env).toBe(process.env as Record<string, string | undefined>)
    expect(ctx.exec).toBe(exec)
  })

  it("accepts explicit overrides", () => {
    const exec = new FakeExec()
    const ctx = makeContext({
      exec,
      cwd: "/tmp",
      env: { FOO: "bar" },
    })

    expect(ctx.cwd).toBe("/tmp")
    expect(ctx.env).toEqual({ FOO: "bar" })
  })
})
