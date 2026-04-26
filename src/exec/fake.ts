import type { Exec, RunInput, RunResult } from "./types"

export class FakeExec implements Exec {
  calls: RunInput[] = []
  private queue: RunResult[] = []

  queueResponse(r: Partial<RunResult>): void {
    this.queue.push({
      exitCode: 0,
      stdout: "",
      stderr: "",
      durationMs: 0,
      timedOut: false,
      ...r,
    })
  }

  async run(input: RunInput): Promise<RunResult> {
    this.calls.push(input)
    const response = this.queue.shift()
    if (!response) {
      throw new Error(`FakeExec: unexpected call ${input.argv.join(" ")}`)
    }
    return response
  }
}
