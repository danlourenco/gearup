import { describe, it, expect, mock, beforeEach } from "bun:test"

const spinnerMock = {
  start: mock<(label: string) => void>(() => undefined),
  message: mock<(label: string) => void>(() => undefined),
  stop: mock<(label: string) => void>(() => undefined),
}

mock.module("@clack/prompts", () => ({
  spinner: () => spinnerMock,
}))

import { createClackReporter } from "./reporter"

beforeEach(() => {
  spinnerMock.start.mockClear()
  spinnerMock.message.mockClear()
  spinnerMock.stop.mockClear()
})

describe("createClackReporter", () => {
  it("calls spinner.start when a step starts", () => {
    const r = createClackReporter()
    r.start("[1/3] jq: checking…")
    expect(spinnerMock.start).toHaveBeenCalledWith("[1/3] jq: checking…")
  })

  it("calls spinner.message on update", () => {
    const r = createClackReporter()
    r.start("[1/3] jq: checking…")
    r.update("[1/3] jq: installing…")
    expect(spinnerMock.message).toHaveBeenCalledWith("[1/3] jq: installing…")
  })

  it("calls spinner.stop on finish", () => {
    const r = createClackReporter()
    r.start("[1/3] jq: checking…")
    r.finish("[1/3] jq: installed")
    expect(spinnerMock.stop).toHaveBeenCalledWith("[1/3] jq: installed")
  })
})
