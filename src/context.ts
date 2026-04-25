import type { Exec } from "./exec/types"

export type Context = {
  exec: Exec
  cwd: string
  env: Record<string, string>
}

type MakeContextInput = {
  exec: Exec
  cwd?: string
  env?: Record<string, string>
}

export function makeContext(input: MakeContextInput): Context {
  return {
    exec: input.exec,
    cwd: input.cwd ?? process.cwd(),
    env: input.env ?? (process.env as Record<string, string>),
  }
}
