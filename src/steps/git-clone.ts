import type { GitCloneStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

function expandHome(p: string, home: string): string {
  if (p.startsWith("~/")) {
    return `${home}/${p.slice(2)}`
  }
  return p
}

export async function checkGitClone(step: GitCloneStep, ctx: Context): Promise<CheckResult> {
  if (step.check) {
    const result = await ctx.exec.run({ argv: [step.check], shell: true })
    return result.exitCode === 0 ? { installed: true } : { installed: false }
  }

  const home = ctx.env.HOME ?? ""
  const dest = expandHome(step.dest, home)
  const result = await ctx.exec.run({ argv: ["test", "-d", dest] })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
