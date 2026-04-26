import fs from "node:fs/promises"
import path from "node:path"
import type { Logger } from "./types"

/**
 * FileLogger writes lines to a file using Bun.file's writer. Buffered, not streamed —
 * each `log()` call accumulates in the writer's internal buffer, flushed on close().
 */
export class FileLogger implements Logger {
  private writer: { write(s: string): number | Promise<number>; end(): number | Promise<number> }
  private closed = false

  constructor(private filePath: string) {
    this.writer = Bun.file(filePath).writer()
  }

  log(line: string): void {
    if (this.closed) {
      throw new Error(`FileLogger: log() called after closed (${this.filePath})`)
    }
    this.writer.write(`${line}\n`)
  }

  path(): string {
    return this.filePath
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    await this.writer.end()
  }
}

/**
 * Convenience: ensure the parent directory exists, then open a FileLogger.
 */
export async function openFileLogger(filePath: string): Promise<FileLogger> {
  await fs.mkdir(path.dirname(filePath), { recursive: true })
  return new FileLogger(filePath)
}
