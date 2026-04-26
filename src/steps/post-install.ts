import type { Context } from "../context"
import type { InstallResult } from "./types"

export async function runPostInstall(
  commands: string[],
  stepName: string,
  ctx: Context,
): Promise<InstallResult> {
  for (let i = 0; i < commands.length; i++) {
    const cmd = commands[i]!
    const result = await ctx.exec.run({ argv: [cmd], shell: true })
    if (result.exitCode !== 0) {
      return {
        ok: false,
        error: `${stepName} post_install[${i}] failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
      }
    }
  }
  return { ok: true }
}
