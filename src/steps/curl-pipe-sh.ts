import type { CurlPipeStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkCurlPipe(step: CurlPipeStep, ctx: Context): Promise<CheckResult> {
  const result = await ctx.exec.run({ argv: [step.check], shell: true })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
