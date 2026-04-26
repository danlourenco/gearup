import * as v from "valibot"
import { step } from "./step"

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
  sources: v.optional(
    v.array(
      v.object({
        path: v.pipe(v.string(), v.minLength(1)),
      }),
    ),
  ),
  extends: v.optional(v.array(v.string())),
  steps: v.optional(v.array(step)),
})

export type Config = v.InferOutput<typeof config>
