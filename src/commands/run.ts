import { defineCommand } from "citty"
import * as clack from "@clack/prompts"
import { loadConfig } from "../config/load"
import { runInstall } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"
import { LoggingExec } from "../exec/logging"
import { openFileLogger } from "../log/file"
import { logFilePath } from "../log/xdg"
import { createClackReporter } from "../ui/reporter"

export const runCommand = defineCommand({
  meta: {
    name: "run",
    description: "Install all configured tools (running each step's install if not already installed)",
  },
  args: {
    config: {
      type: "string",
      description: "Path to config file (JSONC, YAML, or TOML)",
      required: true,
    },
  },
  async run({ args }) {
    const config = await loadConfig(args.config)
    const path = logFilePath(config.name, process.env)
    const logger = await openFileLogger(path)
    const exec = new LoggingExec(new ExecaExec(), logger)
    const ctx = makeContext({ exec, log: logger })

    clack.intro(`gearup run · ${config.name}`)

    try {
      const reporter = createClackReporter()
      const report = await runInstall(config, ctx, reporter)

      if (!report.ok) {
        clack.log.error(`Failed at step: ${report.failedAt}`)
        clack.note(report.error, "error")
        clack.outro(`Log: ${logger.path()}`)
        return 1
      }

      clack.outro(`Done · Log: ${logger.path()}`)
      return 0
    } finally {
      await logger.close()
    }
  },
})
