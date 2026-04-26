import { loadConfig as c12LoadConfig } from "c12"
import * as v from "valibot"
import path from "node:path"
import fs from "node:fs/promises"
import { config as configSchema, type Config } from "../schema"

const KNOWN_EXTENSIONS = [
  ".jsonc",
  ".json",
  ".yaml",
  ".yml",
  ".toml",
  ".ts",
  ".js",
  ".mjs",
  ".cjs",
]

/**
 * Split a user-provided config path into c12's expected (cwd, configFile) pair.
 *
 * c12's `configFile` is a base name without extension; it tries each supported extension
 * in `cwd`. We accept either a path with extension (`backend.jsonc`) or without
 * (`backend`), absolute or relative to process.cwd().
 */
function deriveLoadOptions(configPath: string): { cwd: string; configFile: string } {
  const abs = path.resolve(configPath)
  const dir = path.dirname(abs)
  const base = path.basename(abs)
  const ext = KNOWN_EXTENSIONS.find((e) => base.endsWith(e))
  const stem = ext ? base.slice(0, -ext.length) : base
  return { cwd: dir, configFile: stem }
}

export async function loadConfig(configPath: string): Promise<Config> {
  const abs = path.resolve(configPath)

  // Surface a clearer error when the file doesn't exist — c12's error is opaque.
  // For paths without extension, check that at least one supported extension resolves.
  const hasExtension = KNOWN_EXTENSIONS.some((e) => abs.endsWith(e))
  if (hasExtension) {
    try {
      await fs.access(abs)
    } catch {
      throw new Error(`cannot read config ${abs}: file not found`)
    }
  } else {
    // Try each known extension; fail if none exist.
    let found = false
    for (const ext of KNOWN_EXTENSIONS) {
      try {
        await fs.access(abs + ext)
        found = true
        break
      } catch {
        // try next
      }
    }
    if (!found) {
      throw new Error(`cannot read config ${abs}: file not found (tried ${KNOWN_EXTENSIONS.join(", ")})`)
    }
  }

  const { cwd, configFile } = deriveLoadOptions(configPath)

  // NOTE: c12 auto-resolves extensions ONLY for the entry config (this call's
  // `configFile` parameter). Inside `extends:` arrays, references to non-JS files
  // MUST include their extension — `extends: ["./base.jsonc"]`, not `["./base"]`.
  // Bare names like `"base"` won't resolve and the parent's steps will silently
  // fail to merge. (See tests/fixtures/__bare-name-extends.jsonc test in load.test.ts.)
  const { config: raw } = await c12LoadConfig({
    cwd,
    configFile,
    rcFile: false,
    globalRc: false,
  })

  if (raw == null || (typeof raw === "object" && Object.keys(raw).length === 0)) {
    throw new Error(`config ${abs} loaded as empty — c12 may not have found the file`)
  }

  try {
    return v.parse(configSchema, raw)
  } catch (err) {
    if (err instanceof v.ValiError) {
      const issues = err.issues
        .map(
          (i) =>
            `  - ${i.path?.map((p: v.IssuePathItem) => p.key).join(".") ?? "<root>"}: ${i.message}`,
        )
        .join("\n")
      throw new Error(`config ${abs} failed schema validation:\n${issues}`)
    }
    throw err
  }
}
