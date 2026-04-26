import { join } from "pathe"

const APP_NAME = "gearup"
const LOGS_SUBPATH = `${APP_NAME}/logs`

export function logDir(env: Record<string, string | undefined>): string {
  if (env.XDG_STATE_HOME) {
    return join(env.XDG_STATE_HOME, LOGS_SUBPATH)
  }
  if (env.HOME) {
    return join(env.HOME, ".local/state", LOGS_SUBPATH)
  }
  throw new Error("logDir: neither XDG_STATE_HOME nor HOME is set in the environment")
}

const pad = (n: number) => String(n).padStart(2, "0")

export function timestampedFilename(configName: string, now: Date = new Date()): string {
  const ts =
    `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}` +
    `-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  return `${ts}-${configName}.log`
}

export function logFilePath(
  configName: string,
  env: Record<string, string | undefined>,
  now: Date = new Date(),
): string {
  return join(logDir(env), timestampedFilename(configName, now))
}
