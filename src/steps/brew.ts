import type { BrewStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult, InstallResult } from "./types"

export async function checkBrew(step: BrewStep, ctx: Context): Promise<CheckResult> {
  const input = step.check
    ? { argv: [step.check], shell: true }
    : { argv: ["brew", "list", "--formula", step.formula] }

  const result = await ctx.exec.run(input)
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}

export async function installBrew(step: BrewStep, ctx: Context): Promise<InstallResult> {
  const result = await ctx.exec.run({ argv: ["brew", "install", step.formula] })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `brew install ${step.formula} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
