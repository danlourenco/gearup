import type { Exec, RunInput, RunResult } from "./types"
import type { Logger } from "../log/types"

export class LoggingExec implements Exec {
  constructor(
    private readonly inner: Exec,
    private readonly logger: Logger,
  ) {}

  async run(input: RunInput): Promise<RunResult> {
    this.logger.log(`> ${input.argv.join(" ")}${input.shell ? "  (shell)" : ""}`)
    let result: RunResult
    try {
      result = await this.inner.run(input)
    } catch (err) {
      this.logger.log(`threw: ${err instanceof Error ? err.message : String(err)}`)
      this.logger.log("")
      throw err
    }

    const timeoutSuffix = result.timedOut ? "  (timed out)" : ""
    this.logger.log(`(exit ${result.exitCode}, ${result.durationMs}ms)${timeoutSuffix}`)
    if (result.stdout) {
      this.logger.log("stdout:")
      this.logger.log(result.stdout)
    }
    if (result.stderr) {
      this.logger.log("stderr:")
      this.logger.log(result.stderr)
    }
    this.logger.log("")
    return result
  }
}
