import * as v from "valibot"

const platform = v.object({
  os: v.optional(v.array(v.picklist(["darwin", "linux"]))),
  arch: v.optional(v.array(v.string())),
})

// `check` is optional here so brew/brew-cask/git-clone can fall back to a default;
// curl-pipe-sh and shell override it to required because they have no sensible default.
const baseFields = {
  name: v.pipe(v.string(), v.minLength(1)),
  check: v.optional(v.string()),
  requires_elevation: v.optional(v.boolean()),
  post_install: v.optional(v.array(v.string())),
  platform: v.optional(platform),
}

export const brewStep = v.object({
  type: v.literal("brew"),
  ...baseFields,
  formula: v.pipe(v.string(), v.minLength(1)),
})

export const brewCaskStep = v.object({
  type: v.literal("brew-cask"),
  ...baseFields,
  cask: v.pipe(v.string(), v.minLength(1)),
})

export const curlPipeStep = v.object({
  type: v.literal("curl-pipe-sh"),
  ...baseFields,
  url: v.pipe(v.string(), v.minLength(1)),
  shell: v.optional(v.string()),
  args: v.optional(v.array(v.string())),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const gitCloneStep = v.object({
  type: v.literal("git-clone"),
  ...baseFields,
  repo: v.pipe(v.string(), v.minLength(1)),
  dest: v.pipe(v.string(), v.minLength(1)),
  ref: v.optional(v.string()),
})

export const shellStep = v.object({
  type: v.literal("shell"),
  ...baseFields,
  install: v.pipe(v.string(), v.minLength(1)),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const step = v.variant("type", [
  brewStep,
  brewCaskStep,
  curlPipeStep,
  gitCloneStep,
  shellStep,
])

export type BrewStep = v.InferOutput<typeof brewStep>
export type BrewCaskStep = v.InferOutput<typeof brewCaskStep>
export type CurlPipeStep = v.InferOutput<typeof curlPipeStep>
export type GitCloneStep = v.InferOutput<typeof gitCloneStep>
export type ShellStep = v.InferOutput<typeof shellStep>
export type Step = v.InferOutput<typeof step>
