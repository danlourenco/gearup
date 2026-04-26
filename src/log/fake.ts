import type { Logger } from "./types"

export class FakeLogger implements Logger {
  lines: string[] = []
  closed = false

  constructor(private syntheticPath: string = "/fake/log") {}

  log(line: string): void {
    if (this.closed) {
      throw new Error("FakeLogger: log() called after close()")
    }
    this.lines.push(line)
  }

  path(): string {
    return this.syntheticPath
  }

  async close(): Promise<void> {
    this.closed = true
  }
}
