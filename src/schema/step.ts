import * as v from "valibot"

const platform = v.object({
  os: v.optional(v.array(v.picklist(["darwin", "linux"]))),
  arch: v.optional(v.array(v.string())),
})

// `check` is optional here so brew/brew-cask/git-clone can fall back to a default;
// curl-pipe-sh and shell override it to required because they have no sensible default.
// `name` is NOT in the body — it lives on the parent map's key.
const baseFields = {
  check: v.optional(v.string()),
  requires_elevation: v.optional(v.boolean()),
  post_install: v.optional(v.array(v.string())),
  platform: v.optional(platform),
}

export const brewStepBody = v.object({
  type: v.literal("brew"),
  ...baseFields,
  formula: v.pipe(v.string(), v.minLength(1)),
})

export const brewCaskStepBody = v.object({
  type: v.literal("brew-cask"),
  ...baseFields,
  cask: v.pipe(v.string(), v.minLength(1)),
})

export const curlPipeStepBody = v.object({
  type: v.literal("curl-pipe-sh"),
  ...baseFields,
  url: v.pipe(v.string(), v.url()),
  shell: v.optional(v.picklist(["bash", "sh", "zsh", "fish"])),
  args: v.optional(v.array(v.pipe(v.string(), v.regex(/^\S+$/)))),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const gitCloneStepBody = v.object({
  type: v.literal("git-clone"),
  ...baseFields,
  repo: v.pipe(v.string(), v.minLength(1)),
  dest: v.pipe(v.string(), v.minLength(1)),
  ref: v.optional(v.string()),
})

export const shellStepBody = v.object({
  type: v.literal("shell"),
  ...baseFields,
  install: v.pipe(v.string(), v.minLength(1)),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const stepBody = v.variant("type", [
  brewStepBody,
  brewCaskStepBody,
  curlPipeStepBody,
  gitCloneStepBody,
  shellStepBody,
])

// Internal types: each step has `name` (injected from the Record key during config parsing).
type WithName<T> = T & { name: string }
export type BrewStep = WithName<v.InferOutput<typeof brewStepBody>>
export type BrewCaskStep = WithName<v.InferOutput<typeof brewCaskStepBody>>
export type CurlPipeStep = WithName<v.InferOutput<typeof curlPipeStepBody>>
export type GitCloneStep = WithName<v.InferOutput<typeof gitCloneStepBody>>
export type ShellStep = WithName<v.InferOutput<typeof shellStepBody>>
export type Step = WithName<v.InferOutput<typeof stepBody>>
