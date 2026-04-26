import { describe, it, expect } from "bun:test"
import { makeContext } from "./context"
import { FakeExec } from "./exec/fake"
import { FakeLogger } from "./log/fake"

describe("makeContext", () => {
  it("defaults cwd to process.cwd(), env to process.env, log to a no-op FakeLogger", () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    expect(ctx.cwd).toBe(process.cwd())
    expect(ctx.env).toBe(process.env)
    expect(ctx.exec).toBe(exec)
    expect(typeof ctx.log.log).toBe("function")
    expect(typeof ctx.log.path).toBe("function")
  })

  it("accepts explicit overrides", () => {
    const exec = new FakeExec()
    const log = new FakeLogger("/tmp/explicit.log")
    const ctx = makeContext({
      exec,
      cwd: "/tmp",
      env: { FOO: "bar" },
      log,
    })

    expect(ctx.cwd).toBe("/tmp")
    expect(ctx.env).toEqual({ FOO: "bar" })
    expect(ctx.log).toBe(log)
  })
})
