import fs from "node:fs/promises"
import path from "node:path"
import { EMBEDDED_CONFIGS, readEmbeddedConfig } from "./embedded"

export type ExtractResult = {
  written: string[]
  skipped: string[]
}

export async function extractConfigs(targetDir: string, force: boolean): Promise<ExtractResult> {
  await fs.mkdir(targetDir, { recursive: true })

  const written: string[] = []
  const skipped: string[] = []

  for (const filename of Object.keys(EMBEDDED_CONFIGS)) {
    const dest = path.join(targetDir, filename)

    if (!force) {
      try {
        await fs.access(dest)
        skipped.push(filename)
        continue
      } catch {
        // doesn't exist; fall through to write
      }
    }

    const content = await readEmbeddedConfig(filename)
    await fs.writeFile(dest, content)
    written.push(filename)
  }

  return { written, skipped }
}
