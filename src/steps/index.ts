import type { Step } from "../schema"
import type { Handler } from "./types"
import { checkBrew } from "./brew"
import { checkBrewCask } from "./brew-cask"
import { checkCurlPipe } from "./curl-pipe-sh"
import { checkGitClone } from "./git-clone"
import { checkShell } from "./shell"

export const handlers = {
  brew:           { check: checkBrew },
  "brew-cask":    { check: checkBrewCask },
  "curl-pipe-sh": { check: checkCurlPipe },
  "git-clone":    { check: checkGitClone },
  shell:          { check: checkShell },
} satisfies { [K in Step["type"]]: Handler<Extract<Step, { type: K }>> }

export type Handlers = typeof handlers
export { type Handler, type CheckResult, type InstallResult } from "./types"
