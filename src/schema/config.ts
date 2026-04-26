import * as v from "valibot"
import { stepBody, type Step } from "./step"

// Steps are authored as a Record keyed by name; we transform to Step[] with name injected
// from the key. This shape lets defu (via c12) handle override semantics natively when configs
// are merged through `extends:`.
const stepsRecord = v.pipe(
  v.record(v.string(), stepBody),
  v.transform((rec): Step[] =>
    Object.entries(rec).map(([name, body]) => ({ name, ...body } as Step)),
  ),
)

export const config = v.object({
  version: v.literal(1),
  name: v.pipe(v.string(), v.minLength(1)),
  description: v.optional(v.string()),
  platform: v.optional(
    v.object({
      os: v.optional(v.array(v.picklist(["darwin", "linux"]))),
      arch: v.optional(v.array(v.string())),
    }),
  ),
  elevation: v.optional(
    v.object({
      message: v.string(),
      duration: v.optional(v.string()),
    }),
  ),
  extends: v.optional(v.array(v.string())),
  steps: v.optional(stepsRecord),
})

export type Config = v.InferOutput<typeof config>
