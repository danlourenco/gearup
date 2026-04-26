import * as clack from "@clack/prompts"
import type { ProgressReporter } from "../runner/progress"

/**
 * Build a ProgressReporter that drives a single Clack spinner per step. Steps
 * that arrive before the previous one's `finish` would clobber the spinner;
 * runner.ts is expected to call start/finish in matched pairs.
 */
export function createClackReporter(): ProgressReporter {
  let s: ReturnType<typeof clack.spinner> | null = null

  return {
    start(label) {
      s = clack.spinner()
      s.start(label)
    },
    update(label) {
      s?.message(label)
    },
    finish(label) {
      s?.stop(label)
      s = null
    },
  }
}
