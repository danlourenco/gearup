import { describe, it, expect } from "bun:test"
import { logDir, timestampedFilename, logFilePath } from "./xdg"

describe("logDir", () => {
  it("uses $XDG_STATE_HOME when set", () => {
    const result = logDir({ XDG_STATE_HOME: "/var/lib/state", HOME: "/Users/test" })
    expect(result).toBe("/var/lib/state/gearup/logs")
  })

  it("falls back to $HOME/.local/state when XDG_STATE_HOME is unset", () => {
    const result = logDir({ HOME: "/Users/test" })
    expect(result).toBe("/Users/test/.local/state/gearup/logs")
  })

  it("throws when neither XDG_STATE_HOME nor HOME is set", () => {
    expect(() => logDir({})).toThrow(/HOME/)
  })
})

describe("timestampedFilename", () => {
  it("formats the filename as YYYYMMDD-HHMMSS-<name>.log", () => {
    const fixed = new Date("2026-04-15T21:15:27Z")
    const result = timestampedFilename("Backend", fixed)
    expect(result).toMatch(/^\d{8}-\d{6}-Backend\.log$/)
  })
})

describe("logFilePath", () => {
  it("composes logDir + timestampedFilename", () => {
    const fixed = new Date("2026-04-15T21:15:27Z")
    const result = logFilePath("Backend", { HOME: "/Users/test" }, fixed)
    expect(result).toMatch(
      /^\/Users\/test\/\.local\/state\/gearup\/logs\/\d{8}-\d{6}-Backend\.log$/,
    )
  })
})
