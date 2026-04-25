import { describe, it, expect } from "bun:test"
import * as v from "valibot"
import { step } from "./step"

describe("step schema", () => {
  it("parses a brew step", () => {
    const parsed = v.parse(step, {
      type: "brew",
      name: "jq",
      formula: "jq",
    })
    expect(parsed.type).toBe("brew")
    if (parsed.type === "brew") {
      expect(parsed.formula).toBe("jq")
    }
  })

  it("parses a brew-cask step", () => {
    const parsed = v.parse(step, {
      type: "brew-cask",
      name: "iTerm2",
      cask: "iterm2",
    })
    expect(parsed.type).toBe("brew-cask")
    if (parsed.type === "brew-cask") {
      expect(parsed.cask).toBe("iterm2")
    }
  })

  it("parses a curl-pipe-sh step (check required)", () => {
    const parsed = v.parse(step, {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh without check", () => {
    expect(() =>
      v.parse(step, {
        type: "curl-pipe-sh",
        name: "Homebrew",
        url: "https://example.com/install.sh",
      }),
    ).toThrow()
  })

  it("parses a git-clone step", () => {
    const parsed = v.parse(step, {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    })
    expect(parsed.type).toBe("git-clone")
  })

  it("parses a shell step (check required)", () => {
    const parsed = v.parse(step, {
      type: "shell",
      name: "rust",
      install: "curl ... | sh",
      check: "command -v rustc",
    })
    expect(parsed.type).toBe("shell")
  })

  it("rejects shell step without check", () => {
    expect(() =>
      v.parse(step, {
        type: "shell",
        name: "rust",
        install: "curl ... | sh",
      }),
    ).toThrow()
  })

  it("rejects an unknown step type", () => {
    expect(() =>
      v.parse(step, {
        type: "wat",
        name: "x",
      }),
    ).toThrow()
  })

  it("accepts optional fields: post_install, requires_elevation, check, platform", () => {
    const parsed = v.parse(step, {
      type: "brew",
      name: "colima",
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

import { config } from "./config"

describe("config schema", () => {
  it("parses a minimal config", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "base",
    })
    expect(parsed.name).toBe("base")
    expect(parsed.steps).toBeUndefined()
  })

  it("parses a config with steps", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
      ],
    })
    expect(parsed.steps).toHaveLength(1)
  })

  it("parses a config with extends", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      extends: ["base", "jvm"],
    })
    expect(parsed.extends).toEqual(["base", "jvm"])
  })

  it("parses a config with elevation", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "team",
      elevation: {
        message: "Admin please",
        duration: "180s",
      },
    })
    expect(parsed.elevation?.message).toBe("Admin please")
  })

  it("rejects version !== 1", () => {
    expect(() =>
      v.parse(config, { version: 2, name: "x" }),
    ).toThrow()
  })

  it("rejects a config with no name", () => {
    expect(() =>
      v.parse(config, { version: 1 }),
    ).toThrow()
  })
})
