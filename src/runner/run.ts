import type { Config, Step } from "../schema"
import type { Context } from "../context"
import type { CheckResult, InstallResult } from "../steps/types"
import { handlers } from "../steps"
import { runPostInstall } from "../steps/post-install"
import { acquireElevation } from "../elevation/acquire"
import type { ProgressReporter } from "./progress"
import { NoopReporter } from "./progress"

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

// New types for runInstall reports.
export type StepAction = "installed" | "skipped"

export type InstallStepReport = {
  name: string
  type: Step["type"]
  action: StepAction
}

export type RunReport =
  | {
      ok: true
      configName: string
      steps: InstallStepReport[]
    }
  | {
      ok: false
      configName: string
      steps: InstallStepReport[]
      failedAt: string
      error: string
    }

async function dispatchInstall(step: Step, ctx: Context): Promise<InstallResult> {
  switch (step.type) {
    case "brew":           return handlers.brew.install(step, ctx)
    case "brew-cask":      return handlers["brew-cask"].install(step, ctx)
    case "curl-pipe-sh":   return handlers["curl-pipe-sh"].install(step, ctx)
    case "git-clone":      return handlers["git-clone"].install(step, ctx)
    case "shell":          return handlers.shell.install(step, ctx)
  }
}

async function safeCheck(step: Step, ctx: Context): Promise<{ ok: true; result: CheckResult } | { ok: false; error: string }> {
  try {
    return { ok: true, result: await dispatchCheck(step, ctx) }
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : String(err) }
  }
}

export async function runInstall(
  config: Config,
  ctx: Context,
  progress: ProgressReporter = new NoopReporter(),
): Promise<RunReport> {
  const allSteps = config.steps ?? []
  const completed: InstallStepReport[] = []

  // Partition into elevation-required vs regular, preserving relative order.
  const elevSteps = allSteps.filter((s) => s.requires_elevation === true)
  const regSteps = allSteps.filter((s) => s.requires_elevation !== true)

  // Pre-check: any elevation step that needs install?
  // Deliberate: if a step has requires_elevation: true but config.elevation is
  // absent, no banner is shown — matches Go's runner.go runLive behavior.
  let needsElevation = false
  if (elevSteps.length > 0 && config.elevation) {
    for (const step of elevSteps) {
      const checked = await safeCheck(step, ctx)
      if (!checked.ok) {
        return {
          ok: false, configName: config.name, steps: completed,
          failedAt: step.name, error: `check failed for ${step.name}: ${checked.error}`,
        }
      }
      if (!checked.result.installed) {
        needsElevation = true
        break
      }
    }
  }

  if (needsElevation && config.elevation) {
    const acquired = await acquireElevation(config.elevation)
    if (!acquired.ok) {
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: "elevation", error: acquired.reason,
      }
    }
  }

  const ordered = needsElevation ? [...elevSteps, ...regSteps] : allSteps
  const total = ordered.length

  for (let i = 0; i < ordered.length; i++) {
    const step = ordered[i]!
    const stepNum = i + 1
    const prefix = `[${stepNum}/${total}] ${step.name}`

    progress.start(`${prefix}: checking…`)

    const checked = await safeCheck(step, ctx)
    if (!checked.ok) {
      progress.finish(`${prefix}: check failed`)
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: step.name, error: `check failed for ${step.name}: ${checked.error}`,
      }
    }

    if (checked.result.installed) {
      progress.finish(`${prefix}: already installed`)
      completed.push({ name: step.name, type: step.type, action: "skipped" })
      continue
    }

    progress.update(`${prefix}: installing…`)

    const installResult = await dispatchInstall(step, ctx)
    if (!installResult.ok) {
      progress.finish(`${prefix}: install failed`)
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: step.name, error: installResult.error,
      }
    }

    if (step.post_install && step.post_install.length > 0) {
      progress.update(`${prefix}: post-install…`)
      const postResult = await runPostInstall(step.post_install, step.name, ctx)
      if (!postResult.ok) {
        progress.finish(`${prefix}: post-install failed`)
        return {
          ok: false, configName: config.name, steps: completed,
          failedAt: step.name, error: postResult.error,
        }
      }
    }

    progress.finish(`${prefix}: installed`)
    completed.push({ name: step.name, type: step.type, action: "installed" })
  }

  return { ok: true, configName: config.name, steps: completed }
}
