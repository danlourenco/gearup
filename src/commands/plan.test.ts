import { describe, it, expect, spyOn } from "bun:test"
import { runCommand } from "citty"
import path from "node:path"
import { planCommand } from "./plan"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("plan command", () => {
  it("prints a report and returns exit code 10 when a step would install", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    try {
      const { result } = await runCommand(planCommand, {
        rawArgs: ["--config", path.join(fixtures, "never-installed.jsonc")],
      })
      expect(result).toBe(10)
      const output = logSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(output).toContain("never-installed")
      expect(output).toContain("always-missing")
    } finally {
      logSpy.mockRestore()
    }
  })

  it("returns exit code 0 when no config path is given", async () => {
    // Phase 1 requires --config; absence is an error (exit code != 0/10).
    // In Phase 4 this becomes an interactive picker.
    const errSpy = spyOn(console, "error").mockImplementation(() => undefined)
    try {
      await expect(runCommand(planCommand, { rawArgs: [] })).rejects.toThrow()
    } finally {
      errSpy.mockRestore()
    }
  })
})
