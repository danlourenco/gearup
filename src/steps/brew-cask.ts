import type { BrewCaskStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkBrewCask(step: BrewCaskStep, ctx: Context): Promise<CheckResult> {
  const input = step.check
    ? { argv: [step.check], shell: true }
    : { argv: ["brew", "list", "--cask", step.cask] }

  const result = await ctx.exec.run(input)
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
