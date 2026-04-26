#!/usr/bin/env bun
import { type CommandDef, defineCommand, runMain, runCommand } from "citty"
import * as clack from "@clack/prompts"
import path from "node:path"
import { planCommand } from "./commands/plan"
import { runCommand as gearupRunCommand } from "./commands/run"
import { versionCommand } from "./commands/version"
import { initCommand, userConfigsDir } from "./commands/init"
import { discoverConfigs, pickConfig } from "./ui/picker"
import { extractConfigs } from "./configs/extract"
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
    init: initCommand,
    version: versionCommand,
  },
})

const rawArgs = process.argv.slice(2)

// Dispatch known subcommands via runCommand so numeric exit codes are surfaced.
// citty's runMain discards subcommand return values.
// CommandDef<any> avoids a spurious contravariance error from mismatched ArgsDef shapes.
const subCommands: Record<string, CommandDef<any>> = {
  plan: planCommand,
  run: gearupRunCommand,
  init: initCommand,
  version: versionCommand,
}
const cmdName = rawArgs[0]

const isHelp = rawArgs.includes("--help") || rawArgs.includes("-h")

async function ensureConfigPath(args: string[]): Promise<string[] | null> {
  // If --config is already provided, nothing to do.
  if (args.includes("--config")) return args

  // Discover available configs. Auto-extract embedded defaults on first run.
  const userDir = userConfigsDir(process.env)
  const projectDir = path.resolve(process.cwd(), "configs")

  let configs = await discoverConfigs({ userDir, projectDir })

  if (configs.length === 0) {
    // First run: extract embedded defaults to user dir, then re-discover.
    await extractConfigs(userDir, false)
    configs = await discoverConfigs({ userDir, projectDir })
    if (configs.length === 0) {
      clack.cancel(`No configs found. Try \`gearup init\`.`)
      return null
    }
  }

  const choice = await pickConfig(configs)
  if (choice === null) {
    clack.cancel("Cancelled")
    return null
  }
  return [...args, "--config", choice]
}

async function main() {
  if (cmdName && cmdName in subCommands && !isHelp) {
    let dispatchArgs = rawArgs.slice(1)

    // For plan and run, if --config is missing, present the picker.
    if (cmdName === "plan" || cmdName === "run") {
      const augmented = await ensureConfigPath(dispatchArgs)
      if (augmented === null) {
        process.exit(0)
      }
      dispatchArgs = augmented
    }

    const { result } = await runCommand(subCommands[cmdName]!, { rawArgs: dispatchArgs })
    if (typeof result === "number" && result !== 0) {
      process.exit(result)
    }
  } else {
    runMain(mainCommand)
  }
}

main().catch((err) => {
  console.error(err instanceof Error ? err.message : String(err))
  process.exit(1)
})
