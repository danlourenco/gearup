/**
 * ProgressReporter is a thin event surface the runner uses to inform a UI layer
 * about per-step progress without knowing what the UI looks like.
 *
 * - `start(label)` is called when a step begins (before its `check`).
 * - `update(label)` may be called multiple times during the step (e.g., transitioning
 *   from "checking..." to "installing..." to "post-install...").
 * - `finish(label)` is called once when the step ends (success or failure).
 *
 * Implementations are expected to track a single in-flight step at a time.
 */
export interface ProgressReporter {
  start(label: string): void
  update(label: string): void
  finish(label: string): void
}

/** No-op reporter — used in tests by default. */
export class NoopReporter implements ProgressReporter {
  start(_label: string): void {}
  update(_label: string): void {}
  finish(_label: string): void {}
}

/** Reporter that records every call, for use in tests that want to assert on events. */
export class FakeReporter implements ProgressReporter {
  events: { kind: "start" | "update" | "finish"; label: string }[] = []
  start(label: string): void {
    this.events.push({ kind: "start", label })
  }
  update(label: string): void {
    this.events.push({ kind: "update", label })
  }
  finish(label: string): void {
    this.events.push({ kind: "finish", label })
  }
}
