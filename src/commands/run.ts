import { defineCommand } from "citty"
import { loadConfig } from "../config/load"
import { runInstall, type InstallStepReport } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"

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
    const ctx = makeContext({ exec: new ExecaExec() })
    const report = await runInstall(config, ctx)

    printReport(report.configName, report.steps)

    if (!report.ok) {
      console.error("")
      console.error(`✗ Failed at step: ${report.failedAt}`)
      console.error(`  ${report.error}`)
      return 1
    }

    console.log("")
    console.log("Done.")
    return 0
  },
})

function printReport(configName: string, steps: InstallStepReport[]): void {
  console.log(`CONFIG: ${configName}  (${steps.length} step${steps.length === 1 ? "" : "s"})`)
  console.log("")
  steps.forEach((step, i) => {
    const idx = `[${i + 1}/${steps.length}]`
    const marker = "✓"  // both skipped and installed are successful outcomes
    const label = step.action === "skipped" ? "already installed" : "installed"
    console.log(`  ${marker} ${idx} ${step.name}  ${label}`)
  })
}
