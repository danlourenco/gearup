import { describe, it, expect, mock } from "bun:test"
import { FakeReporter } from "./progress"

// Mock @clack/prompts BEFORE importing modules that use it (acquireElevation).
mock.module("@clack/prompts", () => ({
  confirm: mock(async () => true),
  isCancel: (v: unknown) => v === Symbol.for("clack:cancel"),
  intro: mock(() => undefined),
  outro: mock(() => undefined),
  note: mock(() => undefined),
}))

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

  it("runs steps in declared order when no elevation is needed", async () => {
    const exec = new FakeExec()
    // Pre-check loop runs for elevation steps only:
    //   - elev1 check exit 0 (installed, continue)
    //   - elev2 check exit 0 (installed, loop ends without setting needsElevation)
    exec.queueResponse({ exitCode: 0 })  // pre-check elev1
    exec.queueResponse({ exitCode: 0 })  // pre-check elev2
    // Main loop in declared order: elev1, reg, elev2
    exec.queueResponse({ exitCode: 0 })  // main loop elev1 check (skip)
    exec.queueResponse({ exitCode: 1 })  // main loop reg check (not installed)
    exec.queueResponse({ exitCode: 0 })  // main loop reg install
    exec.queueResponse({ exitCode: 0 })  // main loop elev2 check (skip)

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "ordering",
      elevation: { message: "elev needed" },
      steps: [
        { type: "shell", name: "elev1", install: "true", check: "true", requires_elevation: true },
        { type: "shell", name: "reg", install: "true", check: "false" },
        { type: "shell", name: "elev2", install: "true", check: "true", requires_elevation: true },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    if (report.ok) {
      // Declared order preserved: elev1, reg, elev2
      expect(report.steps.map((s) => s.name)).toEqual(["elev1", "reg", "elev2"])
      expect(report.steps[0]?.action).toBe("skipped")
      expect(report.steps[1]?.action).toBe("installed")
      expect(report.steps[2]?.action).toBe("skipped")
    }
  })

  it("runs elevation steps first when elevation is acquired", async () => {
    // Mock @clack/prompts so acquireElevation returns ok without actually prompting.
    const clack = await import("@clack/prompts")
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => true)

    const exec = new FakeExec()
    // Pre-check: elev1 not installed → needsElevation = true (loop breaks early)
    exec.queueResponse({ exitCode: 1 })  // pre-check elev1 (not installed → break)
    // Main loop in elevation-first order: elev1, reg
    exec.queueResponse({ exitCode: 1 })  // main loop elev1 check (not installed)
    exec.queueResponse({ exitCode: 0 })  // main loop elev1 install
    exec.queueResponse({ exitCode: 1 })  // main loop reg check (not installed)
    exec.queueResponse({ exitCode: 0 })  // main loop reg install

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "elev-first",
      elevation: { message: "elev needed" },
      steps: [
        { type: "shell", name: "reg", install: "true", check: "false" },
        { type: "shell", name: "elev1", install: "true", check: "false", requires_elevation: true },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    if (report.ok) {
      // Elevation-first ordering: elev1 runs before reg (overrides declared order)
      expect(report.steps.map((s) => s.name)).toEqual(["elev1", "reg"])
    }
  })

  it("runs requires_elevation step without prompting when no config.elevation block is set", async () => {
    // No elevation block in config — pre-check is skipped entirely; banner never shown.
    // Main loop runs in declared order (needsElevation stays false).
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // elev step check (not installed)
    exec.queueResponse({ exitCode: 0 })  // elev step install

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "no-elev-block",
      steps: [
        { type: "shell", name: "elev", install: "true", check: "false", requires_elevation: true },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    if (report.ok) {
      expect(report.steps).toHaveLength(1)
      expect(report.steps[0]?.action).toBe("installed")
    }
    expect(exec.calls).toHaveLength(2)  // no extra pre-check call
  })
})

describe("runInstall ProgressReporter integration", () => {
  it("emits start/finish around each step (skip path)", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // step check: installed
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "skip-only",
      steps: [{ type: "brew", name: "jq", formula: "jq" }],
    }

    const reporter = new FakeReporter()
    const report = await runInstall(config, ctx, reporter)

    expect(report.ok).toBe(true)
    expect(reporter.events).toEqual([
      { kind: "start", label: "[1/1] jq: checking…" },
      { kind: "finish", label: "[1/1] jq: already installed" },
    ])
  })

  it("emits start/update/finish around install + post_install", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // check: not installed
    exec.queueResponse({ exitCode: 0 })  // install: ok
    exec.queueResponse({ exitCode: 0 })  // post_install: ok
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "with-post",
      steps: [
        {
          type: "shell",
          name: "rust",
          install: "true",
          check: "false",
          post_install: ["echo done"],
        },
      ],
    }

    const reporter = new FakeReporter()
    await runInstall(config, ctx, reporter)

    const kinds = reporter.events.map((e) => e.kind)
    expect(kinds).toEqual(["start", "update", "update", "finish"])
    expect(reporter.events[0]?.label).toContain("checking")
    expect(reporter.events[1]?.label).toContain("installing")
    expect(reporter.events[2]?.label).toContain("post-install")
    expect(reporter.events[3]?.label).toContain("installed")
  })
})
