// At build time, Bun's bundler sees each `import ... with { type: "file" }` and
// bundles the file contents into the compiled binary. At runtime, the imported
// value is a path string; `Bun.file(path).text()` reads the embedded content.
//
// In dev mode (`bun run`), the path resolves to the source file on disk, so this
// works transparently for both `bun run src/cli.ts` and `bun build --compile`.

import basePath from "../../configs/base.jsonc" with { type: "file" }
import backendPath from "../../configs/backend.jsonc" with { type: "file" }
import frontendPath from "../../configs/frontend.jsonc" with { type: "file" }
import jvmPath from "../../configs/jvm.jsonc" with { type: "file" }
import containersPath from "../../configs/containers.jsonc" with { type: "file" }
import awsK8sPath from "../../configs/aws-k8s.jsonc" with { type: "file" }
import nodePath from "../../configs/node.jsonc" with { type: "file" }

/** Map from config filename → embedded asset path. Order matters: this is the order
 *  they appear in `gearup init` output. */
export const EMBEDDED_CONFIGS: Record<string, string> = {
  "base.jsonc": basePath,
  "backend.jsonc": backendPath,
  "frontend.jsonc": frontendPath,
  "jvm.jsonc": jvmPath,
  "containers.jsonc": containersPath,
  "aws-k8s.jsonc": awsK8sPath,
  "node.jsonc": nodePath,
}

/** Read the contents of an embedded config by filename. */
export async function readEmbeddedConfig(filename: string): Promise<string> {
  const path = EMBEDDED_CONFIGS[filename]
  if (!path) {
    throw new Error(`embedded config not found: ${filename}`)
  }
  return await Bun.file(path).text()
}
