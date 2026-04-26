#!/usr/bin/env bun
import { $ } from "bun"
import fs from "node:fs/promises"
import path from "node:path"
import crypto from "node:crypto"

// Targets to build. macOS-only per the design spec.
const TARGETS = [
  { target: "bun-darwin-arm64", outdir: "dist/darwin-arm64" },
  { target: "bun-darwin-x64", outdir: "dist/darwin-x64" },
] as const

async function sha256(filePath: string): Promise<string> {
  const buf = await fs.readFile(filePath)
  return crypto.createHash("sha256").update(buf).digest("hex")
}

async function main() {
  // Clean dist/
  await fs.rm("dist", { recursive: true, force: true })

  for (const { target, outdir } of TARGETS) {
    console.log(`Building ${target}...`)
    await fs.mkdir(outdir, { recursive: true })
    const outfile = path.join(outdir, "gearup")

    // --external=giget: c12 depends on giget which pulls in node-fetch-native/proxy;
    // that CJS shim has no "fetch" named export, causing bun bundler to error.
    // giget is only used for remote config loading, which gearup doesn't use.
    await $`bun build src/cli.ts --compile --target=${target} --outfile=${outfile} --external=giget`

    const checksum = await sha256(outfile)
    await fs.writeFile(`${outfile}.sha256`, `${checksum}  gearup\n`)

    const stat = await fs.stat(outfile)
    console.log(`  ${outfile} (${(stat.size / 1024 / 1024).toFixed(1)} MB, sha256: ${checksum.slice(0, 16)}...)`)
  }

  console.log("Done.")
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
