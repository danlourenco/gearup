import type { Config, Step } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "../steps/types"
import { handlers } from "../steps"

export type StepStatus = "installed" | "would-install"

export type StepReport = {
  name: string
  type: Step["type"]
  status: StepStatus
}

export type PlanReport = {
  configName: string
  steps: StepReport[]
  exitCode: 0 | 10
}

async function dispatchCheck(step: Step, ctx: Context): Promise<CheckResult> {
  // The discriminated union narrows `step` per branch; each handler accepts its specific variant.
  switch (step.type) {
    case "brew":           return handlers.brew.check(step, ctx)
    case "brew-cask":      return handlers["brew-cask"].check(step, ctx)
    case "curl-pipe-sh":   return handlers["curl-pipe-sh"].check(step, ctx)
    case "git-clone":      return handlers["git-clone"].check(step, ctx)
    case "shell":          return handlers.shell.check(step, ctx)
  }
}

export async function runPlan(config: Config, ctx: Context): Promise<PlanReport> {
  const steps = config.steps ?? []
  const reports: StepReport[] = []

  for (const step of steps) {
    const result = await dispatchCheck(step, ctx)
    reports.push({
      name: step.name,
      type: step.type,
      status: result.installed ? "installed" : "would-install",
    })
  }

  const anyWouldInstall = reports.some((r) => r.status === "would-install")
  return {
    configName: config.name,
    steps: reports,
    exitCode: anyWouldInstall ? 10 : 0,
  }
}
