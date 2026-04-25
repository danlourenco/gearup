import { defineCommand } from "citty"
import { loadConfig } from "../config/load"
import { runPlan, type StepReport } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"

export const planCommand = defineCommand({
  meta: {
    name: "plan",
    description: "Check what would be installed without installing anything",
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
    const report = await runPlan(config, ctx)

    printReport(config.name, report.steps)
    return report.exitCode
  },
})

function printReport(configName: string, steps: StepReport[]): void {
  console.log(`CONFIG: ${configName}  (${steps.length} step${steps.length === 1 ? "" : "s"})`)
  console.log("")
  steps.forEach((step, i) => {
    const idx = `[${i + 1}/${steps.length}]`
    const status = step.status === "installed" ? "✓" : "·"
    const label = step.status === "installed" ? "already installed" : "would install"
    console.log(`  ${status} ${idx} ${step.name}  ${label}`)
  })
  console.log("")
  const wouldInstall = steps.filter((s) => s.status === "would-install").length
  if (wouldInstall === 0) {
    console.log("Machine is up to date.")
  } else {
    console.log(`${wouldInstall} step${wouldInstall === 1 ? "" : "s"} would install.`)
  }
}
