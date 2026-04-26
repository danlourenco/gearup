import type { Exec } from "./exec/types"

export type Context = {
  exec: Exec
  cwd: string
  env: Record<string, string | undefined>
}

type MakeContextInput = {
  exec: Exec
  cwd?: string
  env?: Record<string, string | undefined>
}

export function makeContext(input: MakeContextInput): Context {
  return {
    exec: input.exec,
    cwd: input.cwd ?? process.cwd(),
    env: input.env ?? process.env,
  }
}
