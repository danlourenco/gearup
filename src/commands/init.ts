import { defineCommand } from "citty"
import * as clack from "@clack/prompts"
import path from "node:path"
import { extractConfigs } from "../configs/extract"

export function userConfigsDir(env: Record<string, string | undefined>): string {
  if (env.XDG_CONFIG_HOME) {
    return path.join(env.XDG_CONFIG_HOME, "gearup", "configs")
  }
  if (env.HOME) {
    return path.join(env.HOME, ".config", "gearup", "configs")
  }
  throw new Error("userConfigsDir: neither XDG_CONFIG_HOME nor HOME is set")
}

export const initCommand = defineCommand({
  meta: {
    name: "init",
    description: "Write the embedded default configs to ~/.config/gearup/configs/ (or $XDG_CONFIG_HOME)",
  },
  args: {
    force: {
      type: "boolean",
      description: "Overwrite existing files instead of skipping them",
      default: false,
    },
  },
  async run({ args }) {
    clack.intro("gearup init")

    const targetDir = userConfigsDir(process.env)
    const { written, skipped } = await extractConfigs(targetDir, args.force)

    if (written.length > 0) {
      clack.note(written.join("\n"), `Wrote ${written.length} config${written.length === 1 ? "" : "s"}`)
    }
    if (skipped.length > 0) {
      clack.note(
        skipped.join("\n"),
        `Skipped ${skipped.length} (use --force to overwrite)`,
      )
    }

    clack.outro(`Configs at ${targetDir}`)
    return 0
  },
})
