import * as clack from "@clack/prompts"
import type { Config } from "../schema"

type ElevationConfig = NonNullable<Config["elevation"]>

export type AcquireResult =
  | { ok: true }
  | { ok: false; reason: string }

export async function acquireElevation(cfg: ElevationConfig): Promise<AcquireResult> {
  if (!cfg.message || cfg.message.trim() === "") {
    return { ok: false, reason: "elevation: message is required" }
  }

  // Print the styled banner using Clack's note helper.
  clack.note(cfg.message, "Elevation required")

  const confirmed = await clack.confirm({
    message: "Proceed with elevation-required steps?",
    initialValue: true,
  })

  if (clack.isCancel(confirmed) || confirmed === false) {
    return { ok: false, reason: "elevation aborted by user" }
  }

  return { ok: true }
}
