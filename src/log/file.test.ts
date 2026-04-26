import { describe, it, expect, afterEach } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"
import { FileLogger, openFileLogger } from "./file"

const tmpDir = path.join("/tmp", `gearup-file-logger-test-${process.pid}`)

afterEach(async () => {
  await fs.rm(tmpDir, { recursive: true, force: true })
})

describe("FileLogger", () => {
  it("creates the parent directory if it doesn't exist and writes lines to the file", async () => {
    const filePath = path.join(tmpDir, "subdir", "test.log")
    const logger = await openFileLogger(filePath)

    logger.log("first")
    logger.log("second")
    await logger.close()

    const contents = await fs.readFile(filePath, "utf8")
    expect(contents).toBe("first\nsecond\n")
  })

  it("path() returns the file path", async () => {
    const filePath = path.join(tmpDir, "p.log")
    const logger = await openFileLogger(filePath)
    try {
      expect(logger.path()).toBe(filePath)
    } finally {
      await logger.close()
    }
  })

  it("close() is idempotent", async () => {
    const filePath = path.join(tmpDir, "i.log")
    const logger = await openFileLogger(filePath)
    await logger.close()
    await logger.close()  // must not throw
    expect(true).toBe(true)
  })

  it("log() after close() throws", async () => {
    const filePath = path.join(tmpDir, "afterclose.log")
    const logger = await openFileLogger(filePath)
    await logger.close()
    expect(() => logger.log("nope")).toThrow(/closed/)
  })
})
