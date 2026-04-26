export interface Logger {
  /** Append a line of text to the log. The implementation adds the trailing newline. */
  log(line: string): void
  /** Path to the underlying log destination, for printing on failure. */
  path(): string
  /** Flush and close. Idempotent. */
  close(): Promise<void>
}
