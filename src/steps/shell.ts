import type { ShellStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult, InstallResult } from "./types"

export async function checkShell(step: ShellStep, ctx: Context): Promise<CheckResult> {
  const result = await ctx.exec.run({ argv: [step.check], shell: true })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}

export async function installShell(step: ShellStep, ctx: Context): Promise<InstallResult> {
  const result = await ctx.exec.run({ argv: [step.install], shell: true })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `shell ${step.name} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
