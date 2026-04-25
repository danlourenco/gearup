import type { Step } from "../schema"
import type { Context } from "../context"

export type CheckResult =
  | { installed: true }
  | { installed: false; reason?: string }

export type InstallResult =
  | { ok: true }
  | { ok: false; error: string }

export type Handler<S extends Step> = {
  check: (step: S, ctx: Context) => Promise<CheckResult>
  install?: (step: S, ctx: Context) => Promise<InstallResult>
}
