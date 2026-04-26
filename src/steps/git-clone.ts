import type { GitCloneStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult, InstallResult } from "./types"
import path from "node:path"

function expandHome(p: string, home: string): string {
  if (p === "~") return home
  if (p.startsWith("~/")) return `${home}/${p.slice(2)}`
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

export async function installGitClone(step: GitCloneStep, ctx: Context): Promise<InstallResult> {
  const home = ctx.env.HOME ?? ""
  const dest = expandHome(step.dest, home)
  const parent = path.dirname(dest)

  const mkdirResult = await ctx.exec.run({ argv: ["mkdir", "-p", parent] })
  if (mkdirResult.exitCode !== 0) {
    return {
      ok: false,
      error: `git-clone ${step.name}: create parent dir ${parent} failed (exit ${mkdirResult.exitCode}): ${mkdirResult.stderr.trim()}`,
    }
  }

  const argv = step.ref
    ? ["git", "clone", "--branch", step.ref, step.repo, dest]
    : ["git", "clone", step.repo, dest]

  const cloneResult = await ctx.exec.run({ argv })
  if (cloneResult.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `git clone ${step.name} failed (exit ${cloneResult.exitCode}): ${cloneResult.stderr.trim()}`,
  }
}
