import type { Step } from "../schema"
import type { Handler } from "./types"
import { checkBrew, installBrew } from "./brew"
import { checkBrewCask, installBrewCask } from "./brew-cask"
import { checkCurlPipe, installCurlPipe } from "./curl-pipe-sh"
import { checkGitClone, installGitClone } from "./git-clone"
import { checkShell, installShell } from "./shell"

export const handlers = {
  brew:           { check: checkBrew,     install: installBrew     },
  "brew-cask":    { check: checkBrewCask, install: installBrewCask },
  "curl-pipe-sh": { check: checkCurlPipe, install: installCurlPipe },
  "git-clone":    { check: checkGitClone, install: installGitClone },
  shell:          { check: checkShell,    install: installShell    },
} satisfies { [K in Step["type"]]: Handler<Extract<Step, { type: K }>> }

export type Handlers = typeof handlers
export { type Handler, type CheckResult, type InstallResult } from "./types"
