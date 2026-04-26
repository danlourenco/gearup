import fs from "node:fs/promises"
import path from "node:path"
import { parseJSONC } from "confbox/jsonc"
import { parseYAML } from "confbox/yaml"
import { parseTOML } from "confbox/toml"
import * as clack from "@clack/prompts"

export type ConfigSource = "user" | "project"

export type DiscoveredConfig = {
  name: string
  description?: string
  path: string
  source: ConfigSource
}

const CONFIG_EXTENSIONS = [".jsonc", ".json", ".yaml", ".yml", ".toml"]

type DiscoveryDirs = {
  userDir: string
  projectDir: string
}

/**
 * Discover configs by scanning two directories. The picker UX shows a flat union;
 * on name collision, the project-local copy wins (it's closer to the user's pwd
 * and presumably more relevant in-context).
 *
 * Files are parsed lightly: we read `name` and `description` without resolving
 * `extends:` (that's expensive; pickers should be fast). Files that fail to parse
 * are silently skipped so one broken file doesn't break the whole picker.
 */
export async function discoverConfigs(dirs: DiscoveryDirs): Promise<DiscoveredConfig[]> {
  const userConfigs = await scanDir(dirs.userDir, "user")
  const projectConfigs = await scanDir(dirs.projectDir, "project")

  const seen = new Set<string>()
  const result: DiscoveredConfig[] = []

  for (const c of projectConfigs) {
    if (!seen.has(c.name)) {
      seen.add(c.name)
      result.push(c)
    }
  }
  for (const c of userConfigs) {
    if (!seen.has(c.name)) {
      seen.add(c.name)
      result.push(c)
    }
  }

  return result
}

async function scanDir(dir: string, source: ConfigSource): Promise<DiscoveredConfig[]> {
  let entries: string[]
  try {
    entries = await fs.readdir(dir)
  } catch {
    return []
  }

  const configs: DiscoveredConfig[] = []
  for (const entry of entries) {
    const ext = CONFIG_EXTENSIONS.find((e) => entry.toLowerCase().endsWith(e))
    if (!ext) continue

    const filePath = path.join(dir, entry)
    try {
      const text = await fs.readFile(filePath, "utf8")
      const parsed = parseByExt(ext, text) as { name?: unknown; description?: unknown }
      if (typeof parsed.name === "string") {
        configs.push({
          name: parsed.name,
          description: typeof parsed.description === "string" ? parsed.description : undefined,
          path: filePath,
          source,
        })
      }
    } catch {
      // skip broken files
    }
  }
  return configs
}

function parseByExt(ext: string, text: string): unknown {
  switch (ext) {
    case ".jsonc":
    case ".json":
      return parseJSONC(text)
    case ".yaml":
    case ".yml":
      return parseYAML(text)
    case ".toml":
      return parseTOML(text)
  }
  return null
}

/**
 * Show a Clack select prompt of available configs. Returns the chosen config's
 * path, or null if the user cancels (Ctrl-C / Escape).
 *
 * If exactly one config is available, returns its path immediately without
 * prompting — picking from a list of one is wasteful.
 */
export async function pickConfig(configs: DiscoveredConfig[]): Promise<string | null> {
  if (configs.length === 0) {
    throw new Error("no configs to pick from")
  }
  if (configs.length === 1) {
    return configs[0]!.path
  }

  const choice = await clack.select({
    message: "Pick a config",
    options: configs.map((c) => ({
      label: c.name,
      value: c.path,
      hint: c.description
        ? `${c.description} [${c.source}]`
        : `[${c.source}]`,
    })),
  })

  if (clack.isCancel(choice)) {
    return null
  }
  return choice as string
}
