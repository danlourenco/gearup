import * as v from "valibot"
import path from "node:path"
import fs from "node:fs/promises"
import { parseJSONC } from "confbox/jsonc"
import { parseYAML } from "confbox/yaml"
import { parseTOML } from "confbox/toml"
import { config as configSchema, type Config } from "../schema"

export async function loadConfig(configPath: string): Promise<Config> {
  const abs = path.resolve(configPath)

  let text: string
  try {
    text = await fs.readFile(abs, "utf8")
  } catch (err) {
    throw new Error(`cannot read config ${abs}: ${(err as Error).message}`)
  }

  const raw = parseByExtension(abs, text)

  try {
    return v.parse(configSchema, raw)
  } catch (err) {
    if (err instanceof v.ValiError) {
      const issues = err.issues
        .map((i) => `  - ${i.path?.map((p: v.IssuePathItem) => p.key).join(".") ?? "<root>"}: ${i.message}`)
        .join("\n")
      throw new Error(`config ${abs} failed schema validation:\n${issues}`)
    }
    throw err
  }
}

function parseByExtension(filePath: string, text: string): unknown {
  const ext = path.extname(filePath).toLowerCase()
  switch (ext) {
    case ".jsonc":
    case ".json":
      return parseJSONC(text)
    case ".yaml":
    case ".yml":
      return parseYAML(text)
    case ".toml":
      return parseTOML(text)
    default:
      throw new Error(`unsupported config extension "${ext}" (${filePath})`)
  }
}
