import { defineCommand } from "citty"
import { loadConfig } from "../config/load"
import { runInstall, type InstallStepReport } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"
import { LoggingExec } from "../exec/logging"
import { openFileLogger } from "../log/file"
import { logFilePath } from "../log/xdg"

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

    try {
      const report = await runInstall(config, ctx)

      printReport(report.configName, report.steps)

      if (!report.ok) {
        console.error("")
        console.error(`✗ Failed at step: ${report.failedAt}`)
        console.error(`  ${report.error}`)
        console.error("")
        console.error(`Log: ${logger.path()}`)
        return 1
      }

      console.log("")
      console.log("Done.")
      console.log(`Log: ${logger.path()}`)
      return 0
    } finally {
      await logger.close()
    }
  },
})

function printReport(configName: string, steps: InstallStepReport[]): void {
  console.log(`CONFIG: ${configName}  (${steps.length} step${steps.length === 1 ? "" : "s"})`)
  console.log("")
  steps.forEach((step, i) => {
    const idx = `[${i + 1}/${steps.length}]`
    const marker = "✓"
    const label = step.action === "skipped" ? "already installed" : "installed"
    console.log(`  ${marker} ${idx} ${step.name}  ${label}`)
  })
}
