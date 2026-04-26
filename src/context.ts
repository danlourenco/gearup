import type { Exec } from "./exec/types"
import type { Logger } from "./log/types"
import { FakeLogger } from "./log/fake"

export type Context = {
  exec: Exec
  cwd: string
  env: Record<string, string | undefined>
  log: Logger
}

type MakeContextInput = {
  exec: Exec
  cwd?: string
  env?: Record<string, string | undefined>
  log?: Logger
}

export function makeContext(input: MakeContextInput): Context {
  return {
    exec: input.exec,
    cwd: input.cwd ?? process.cwd(),
    env: input.env ?? process.env,
    log: input.log ?? new FakeLogger("/dev/null"),
  }
}
