import type { CurlPipeStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult, InstallResult } from "./types"

// Shell-quote a single argument so any metacharacters inside it become literal.
// Wraps in single quotes; embedded single quotes are escaped via the standard '\'' trick.
function shellQuote(s: string): string {
  return `'${s.replace(/'/g, "'\\''")}'`
}

export async function checkCurlPipe(step: CurlPipeStep, ctx: Context): Promise<CheckResult> {
  const result = await ctx.exec.run({ argv: [step.check], shell: true })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}

export async function installCurlPipe(step: CurlPipeStep, ctx: Context): Promise<InstallResult> {
  const shell = step.shell ?? "bash"
  let cmd = `curl -fsSL ${shellQuote(step.url)} | ${shellQuote(shell)}`
  if (step.args && step.args.length > 0) {
    cmd = `${cmd} -s -- ${step.args.map(shellQuote).join(" ")}`
  }

  const result = await ctx.exec.run({ argv: [cmd], shell: true })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `curl-pipe-sh ${step.name} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
