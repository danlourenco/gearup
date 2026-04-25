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
