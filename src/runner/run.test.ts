import { describe, it, expect } from "bun:test"
import { runPlan } from "./run"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { Config } from "../schema"

describe("runPlan", () => {
  it("reports each step as either installed or would-install based on check exit code", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // brew installed
    exec.queueResponse({ exitCode: 1 })  // shell not installed
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "mixed",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        { type: "shell", name: "rust", install: "curl ... | sh", check: "command -v rustc" },
      ],
    }

    const report = await runPlan(config, ctx)

    expect(report.steps).toHaveLength(2)
    expect(report.steps[0]).toEqual({ name: "jq", type: "brew", status: "installed" })
    expect(report.steps[1]).toEqual({ name: "rust", type: "shell", status: "would-install" })
    expect(report.exitCode).toBe(10)
  })

  it("returns exit 0 when every step is installed", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "all-good",
      steps: [{ type: "brew", name: "jq", formula: "jq" }],
    }

    const report = await runPlan(config, ctx)

    expect(report.exitCode).toBe(0)
    expect(report.steps[0]?.status).toBe("installed")
  })

  it("returns exit 0 on a config with no steps", async () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    const config: Config = { version: 1, name: "empty" }
    const report = await runPlan(config, ctx)

    expect(report.exitCode).toBe(0)
    expect(report.steps).toHaveLength(0)
  })
})
