import { describe, it, expect } from "bun:test"
import { handlers } from "./index"

describe("handlers registry", () => {
  it("has a check function for every step type", () => {
    const expectedTypes = ["brew", "brew-cask", "curl-pipe-sh", "git-clone", "shell"]
    for (const type of expectedTypes) {
      expect(handlers).toHaveProperty(type)
      expect(typeof (handlers as Record<string, { check: unknown }>)[type]?.check).toBe("function")
    }
  })

  it("has an install function for every step type", () => {
    const expectedTypes = ["brew", "brew-cask", "curl-pipe-sh", "git-clone", "shell"]
    for (const type of expectedTypes) {
      expect(typeof (handlers as Record<string, { install: unknown }>)[type]?.install).toBe("function")
    }
  })
})
