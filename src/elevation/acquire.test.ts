import { describe, it, expect, mock } from "bun:test"

// Mock @clack/prompts BEFORE importing acquire so the import sees the mock.
mock.module("@clack/prompts", () => ({
  confirm: mock(async () => true),
  isCancel: (v: unknown) => v === Symbol.for("clack:cancel"),
  intro: mock(() => undefined),
  outro: mock(() => undefined),
  note: mock(() => undefined),
}))

import { acquireElevation } from "./acquire"
import * as clack from "@clack/prompts"

describe("acquireElevation", () => {
  it("returns ok when the user confirms", async () => {
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => true)

    const result = await acquireElevation({ message: "Need admin", duration: "180s" })

    expect(result.ok).toBe(true)
    expect(clack.note).toHaveBeenCalled()       // banner printed
    expect(clack.confirm).toHaveBeenCalled()    // confirm prompted
  })

  it("returns aborted when the user declines", async () => {
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => false)

    const result = await acquireElevation({ message: "Need admin" })

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.reason).toContain("aborted")
    }
  })

  it("returns aborted when the user cancels (Ctrl-C)", async () => {
    const cancelSymbol = Symbol.for("clack:cancel")
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => cancelSymbol)

    const result = await acquireElevation({ message: "Need admin" })

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.reason).toContain("aborted")
    }
  })

  it("rejects an empty elevation message", async () => {
    const result = await acquireElevation({ message: "" })

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.reason).toContain("message is required")
    }
  })
})
