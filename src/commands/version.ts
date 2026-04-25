import { defineCommand } from "citty"
import pkg from "../../package.json" with { type: "json" }

export const versionCommand = defineCommand({
  meta: {
    name: "version",
    description: "Print the gearup version",
  },
  run() {
    console.log(pkg.version)
  },
})
