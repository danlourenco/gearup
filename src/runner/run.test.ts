import { describe, it, expect } from "bun:test"
import { runPlan, runInstall } from "./run"
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

describe("runInstall", () => {
  it("checks each step; skips installed; installs missing; runs post_install on success", async () => {
    const exec = new FakeExec()
    // step 1 (jq, brew): check exit 0 → installed, skip install
    exec.queueResponse({ exitCode: 0 })
    // step 2 (rust, shell): check exit 1 → not installed; install exit 0; post_install command exit 0
    exec.queueResponse({ exitCode: 1 })  // check
    exec.queueResponse({ exitCode: 0 })  // install
    exec.queueResponse({ exitCode: 0 })  // post_install[0]

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "mixed",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        {
          type: "shell",
          name: "rust",
          install: "curl ... | sh",
          check: "command -v rustc",
          post_install: ["rustup default stable"],
        },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    if (report.ok) {
      expect(report.steps).toHaveLength(2)
      expect(report.steps[0]).toEqual({ name: "jq", type: "brew", action: "skipped" })
      expect(report.steps[1]).toEqual({ name: "rust", type: "shell", action: "installed" })
    }
  })

  it("fails fast on install failure and reports which step", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // jq check: not installed
    exec.queueResponse({ exitCode: 1, stderr: "boom" })  // jq install: fail

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "broken",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        { type: "brew", name: "git", formula: "git" },  // never reached
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(false)
    if (!report.ok) {
      expect(report.failedAt).toBe("jq")
      expect(report.error).toContain("brew install jq")
    }
    expect(exec.calls).toHaveLength(2)
  })

  it("fails fast on check failure", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // first step ok, installed
    // For step 2, don't queue → FakeExec throws

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "check-fail",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        { type: "brew", name: "git", formula: "git" },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(false)
    if (!report.ok) {
      expect(report.failedAt).toBe("git")
      expect(report.error).toContain("unexpected call")
    }
  })

  it("returns ok with empty steps when config has no steps", async () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    const config: Config = { version: 1, name: "empty" }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    if (report.ok) {
      expect(report.steps).toHaveLength(0)
    }
  })

  it("fails fast when post_install fails (and reports which command)", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // check: not installed
    exec.queueResponse({ exitCode: 0 })  // install: ok
    exec.queueResponse({ exitCode: 1, stderr: "post boom" })  // post_install[0]: fail

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "post-fail",
      steps: [
        {
          type: "shell",
          name: "thing",
          install: "true",
          check: "false",
          post_install: ["false"],
        },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(false)
    if (!report.ok) {
      expect(report.failedAt).toBe("thing")
      expect(report.error).toContain("post_install")
    }
  })
})
