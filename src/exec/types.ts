export type RunInput = {
  argv: string[]
  shell?: boolean
  cwd?: string
  env?: Record<string, string>
  stdin?: string
  timeout?: number
}

export type RunResult = {
  exitCode: number
  stdout: string
  stderr: string
  durationMs: number
  timedOut: boolean
}

export interface Exec {
  run(input: RunInput): Promise<RunResult>
}
