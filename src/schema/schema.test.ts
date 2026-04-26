import { describe, it, expect } from "bun:test"
import * as v from "valibot"
import { stepBody } from "./step"
import { config } from "./config"

describe("step body schema", () => {
  it("parses a brew step body (no name field)", () => {
    const parsed = v.parse(stepBody, { type: "brew", formula: "jq" })
    expect(parsed.type).toBe("brew")
    if (parsed.type === "brew") {
      expect(parsed.formula).toBe("jq")
    }
  })

  it("parses a brew-cask step body", () => {
    const parsed = v.parse(stepBody, { type: "brew-cask", cask: "iterm2" })
    expect(parsed.type).toBe("brew-cask")
  })

  it("parses a curl-pipe-sh step body with valid URL", () => {
    const parsed = v.parse(stepBody, {
      type: "curl-pipe-sh",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh with non-URL string for url", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "not a url",
        check: "command -v brew",
      }),
    ).toThrow()
  })

  it("rejects curl-pipe-sh with disallowed shell", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "https://example.com/install.sh",
        shell: "rm",
        check: "command -v brew",
      }),
    ).toThrow()
  })

  it("accepts curl-pipe-sh with allowed shell", () => {
    const parsed = v.parse(stepBody, {
      type: "curl-pipe-sh",
      url: "https://example.com/install.sh",
      shell: "sh",
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh args containing whitespace within an arg", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "https://example.com/install.sh",
        args: ["valid", "has space"],
        check: "command -v brew",
      }),
    ).toThrow()
  })

  it("accepts curl-pipe-sh args that are individually whitespace-free", () => {
    const parsed = v.parse(stepBody, {
      type: "curl-pipe-sh",
      url: "https://example.com/install.sh",
      args: ["-y", "--default-toolchain", "stable"],
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh without check", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "https://example.com/install.sh",
      }),
    ).toThrow()
  })

  it("parses a git-clone step body", () => {
    const parsed = v.parse(stepBody, {
      type: "git-clone",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    })
    expect(parsed.type).toBe("git-clone")
  })

  it("parses a shell step body with required check", () => {
    const parsed = v.parse(stepBody, {
      type: "shell",
      install: "curl ... | sh",
      check: "command -v rustc",
    })
    expect(parsed.type).toBe("shell")
  })

  it("rejects shell step without check", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "shell",
        install: "curl ... | sh",
      }),
    ).toThrow()
  })

  it("rejects an unknown step type", () => {
    expect(() => v.parse(stepBody, { type: "wat" })).toThrow()
  })

  it("accepts optional fields: post_install, requires_elevation, check, platform", () => {
    const parsed = v.parse(stepBody, {
      type: "brew",
      formula: "colima",
      check: "command -v colima",
      requires_elevation: false,
      post_install: ["colima start --cpu 4 --memory 8"],
      platform: { os: ["darwin"] },
    })
    expect(parsed.type).toBe("brew")
    if (parsed.type === "brew") {
      expect(parsed.post_install).toEqual(["colima start --cpu 4 --memory 8"])
    }
  })
})

describe("config schema", () => {
  it("parses a minimal config", () => {
    const parsed = v.parse(config, { version: 1, name: "base" })
    expect(parsed.name).toBe("base")
    expect(parsed.steps).toBeUndefined()
  })

  it("parses a config with steps as a Record (transformed to Step[] with name injected)", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      steps: {
        jq: { type: "brew", formula: "jq" },
        Git: { type: "brew", formula: "git" },
      },
    })
    expect(parsed.steps).toHaveLength(2)
    expect(parsed.steps?.[0]).toEqual({ type: "brew", name: "jq", formula: "jq" })
    expect(parsed.steps?.[1]).toEqual({ type: "brew", name: "Git", formula: "git" })
  })

  it("parses a config with extends", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      extends: ["./base", "./jvm"],
    })
    expect(parsed.extends).toEqual(["./base", "./jvm"])
  })

  it("parses a config with elevation", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "team",
      elevation: { message: "Admin please", duration: "180s" },
    })
    expect(parsed.elevation?.message).toBe("Admin please")
  })

  it("parses a config with both extends and steps coexisting", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "full",
      extends: ["./base"],
      steps: { jq: { type: "brew", formula: "jq" } },
    })
    expect(parsed.extends).toHaveLength(1)
    expect(parsed.steps).toHaveLength(1)
  })

  it("rejects an elevation block missing message", () => {
    expect(() => v.parse(config, { version: 1, name: "x", elevation: {} })).toThrow()
  })

  it("rejects version !== 1", () => {
    expect(() => v.parse(config, { version: 2, name: "x" })).toThrow()
  })

  it("rejects a config with no name", () => {
    expect(() => v.parse(config, { version: 1 })).toThrow()
  })

  it("silently drops the removed `sources` field (Valibot's v.object is non-strict)", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "x",
      sources: [{ path: "./somewhere" }],
    } as never)
    expect(parsed.name).toBe("x")
    expect((parsed as Record<string, unknown>).sources).toBeUndefined()
  })
})
