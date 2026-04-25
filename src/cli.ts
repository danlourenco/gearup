#!/usr/bin/env bun
import { defineCommand, runMain, runCommand } from "citty"
import { planCommand } from "./commands/plan"
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
    version: versionCommand,
  },
})

const rawArgs = process.argv.slice(2)

// For the plan subcommand, capture its numeric exit code and propagate it.
// citty's runMain discards subcommand return values, so we run it directly.
if (rawArgs[0] === "plan") {
  runCommand(planCommand, { rawArgs: rawArgs.slice(1) })
    .then(({ result }) => {
      if (typeof result === "number" && result !== 0) {
        process.exit(result)
      }
    })
    .catch((err) => {
      console.error(err.message)
      process.exit(1)
    })
} else {
  runMain(mainCommand)
}
