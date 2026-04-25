import { execa } from "execa"
import type { Exec, RunInput, RunResult } from "./types"

export class ExecaExec implements Exec {
  async run(input: RunInput): Promise<RunResult> {
    const start = performance.now()
    const [cmd, ...args] = input.shell
      ? [input.argv.join(" ")]
      : input.argv

    if (!cmd) {
      throw new Error("ExecaExec: empty argv")
    }

    const result = await execa(cmd, args, {
      cwd: input.cwd,
      env: input.env,
      input: input.stdin,
      timeout: input.timeout ?? 60_000,
      shell: input.shell ? "/bin/sh" : false,
      reject: false,
    })

    return {
      exitCode: result.exitCode ?? 0,
      stdout: typeof result.stdout === "string" ? result.stdout : "",
      stderr: typeof result.stderr === "string" ? result.stderr : "",
      durationMs: Math.round(performance.now() - start),
      timedOut: result.timedOut === true,
    }
  }
}
