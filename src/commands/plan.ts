import { defineCommand } from "citty"
import * as clack from "@clack/prompts"
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

    clack.intro(`gearup plan · ${config.name}`)
    const report = await runPlan(config, ctx)

    const lines = report.steps.map((s, i) => {
      const idx = `[${i + 1}/${report.steps.length}]`
      const marker = s.status === "installed" ? "✓" : "·"
      const label = s.status === "installed" ? "already installed" : "would install"
      return `${marker} ${idx} ${s.name}  ${label}`
    })
    if (lines.length > 0) {
      clack.note(lines.join("\n"), `${report.steps.length} step${report.steps.length === 1 ? "" : "s"}`)
    }

    const wouldInstall = report.steps.filter((s) => s.status === "would-install").length
    if (wouldInstall === 0) {
      clack.outro("Machine is up to date")
    } else {
      clack.outro(`${wouldInstall} step${wouldInstall === 1 ? "" : "s"} would install`)
    }
    return report.exitCode
  },
})

export type { StepReport }
