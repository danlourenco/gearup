import type { ShellStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkShell(step: ShellStep, ctx: Context): Promise<CheckResult> {
  const result = await ctx.exec.run({ argv: [step.check], shell: true })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
