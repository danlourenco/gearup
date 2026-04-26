#!/usr/bin/env bun
import { type CommandDef, defineCommand, runMain, runCommand } from "citty"
import { planCommand } from "./commands/plan"
import { runCommand as gearupRunCommand } from "./commands/run"
import { versionCommand } from "./commands/version"
import pkg from "../package.json" with { type: "json" }

const mainCommand = defineCommand({
  meta: {
    name: "gearup",
    version: pkg.version,
    description: "Config-driven macOS developer-machine bootstrap",
  },
  subCommands: {
    plan: planCommand,
    run: gearupRunCommand,
    version: versionCommand,
  },
})

const rawArgs = process.argv.slice(2)

// Dispatch known subcommands via runCommand so numeric exit codes are surfaced.
// citty's runMain discards subcommand return values, so it is reserved for
// meta paths (--help, --version, unknown args) that only need help rendering.
// CommandDef<any> avoids a spurious contravariance error from mismatched ArgsDef shapes.
const subCommands: Record<string, CommandDef<any>> = {
  plan: planCommand,
  run: gearupRunCommand,
  version: versionCommand,
}
const cmdName = rawArgs[0]

const isHelp = rawArgs.includes("--help") || rawArgs.includes("-h")

if (cmdName && cmdName in subCommands && !isHelp) {
  runCommand(subCommands[cmdName]!, { rawArgs: rawArgs.slice(1) })
    .then(({ result }) => {
      if (typeof result === "number" && result !== 0) process.exit(result)
    })
    .catch((err) => {
      console.error(err instanceof Error ? err.message : String(err))
      process.exit(1)
    })
} else {
  runMain(mainCommand)
}
