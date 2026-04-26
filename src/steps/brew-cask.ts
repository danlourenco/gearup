import type { BrewCaskStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult, InstallResult } from "./types"

export async function checkBrewCask(step: BrewCaskStep, ctx: Context): Promise<CheckResult> {
  const input = step.check
    ? { argv: [step.check], shell: true }
    : { argv: ["brew", "list", "--cask", step.cask] }

  const result = await ctx.exec.run(input)
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}

export async function installBrewCask(step: BrewCaskStep, ctx: Context): Promise<InstallResult> {
  const result = await ctx.exec.run({ argv: ["brew", "install", "--cask", step.cask] })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `brew install --cask ${step.cask} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
