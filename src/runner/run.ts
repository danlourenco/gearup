import type { Config, Step } from "../schema"
import type { Context } from "../context"
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

export async function runPlan(config: Config, ctx: Context): Promise<PlanReport> {
  const steps = config.steps ?? []
  const reports: StepReport[] = []

  for (const step of steps) {
    const handler = handlers[step.type] as { check: (s: typeof step, c: Context) => Promise<{ installed: boolean }> }
    const result = await handler.check(step, ctx)
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
