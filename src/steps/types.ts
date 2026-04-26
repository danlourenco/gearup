import type { Step } from "../schema"
import type { Context } from "../context"

export type CheckResult =
  | { installed: true }
  // `reason?` is Phase 2 scaffolding for surfacing why a step is not installed.
  | { installed: false; reason?: string }

export type InstallResult =
  | { ok: true }
  | { ok: false; error: string }

export type Handler<S extends Step> = {
  check: (step: S, ctx: Context) => Promise<CheckResult>
  // `install?` is Phase 2 scaffolding; Phase 1 only implements `check`.
  install?: (step: S, ctx: Context) => Promise<InstallResult>
}
