// A hand-written declaration of the kit that `claude plugin test` provides as 'claude-code/testing' (793e8b). Claude
// Code writes the real type files only in an interactive session, so `tsc --noEmit` (targ lint-js) checks the mod's
// test files against this declaration: it types the parts of the kit the tests use and leaves the hook context and the
// event shapes open (`any`), as the tests already pass them.
declare module 'claude-code/testing' {
  interface Matchers {
    not: Matchers
    toBe(expected: unknown): void
    toEqual(expected: unknown): void
    toContain(expected: unknown): void
    toMatch(expected: RegExp | string): void
    toBeGreaterThan(expected: number): void
    toBeLessThanOrEqual(expected: number): void
    toBeUndefined(): void
    toThrow(expected?: RegExp | string): void
  }

  export interface MockClock {
    now(): number | Promise<number>
    advance(milliseconds: number): Promise<void>
    set(milliseconds: number): Promise<void>
    settle(): Promise<void>
    sleep(milliseconds: number): Promise<void>
  }

  // The driver `$` of a test calls the events of the mod (`$.prompt.submit(event)`); `on` registers the stubs that stand
  // for the engine beneath it. Their shapes are the platform's and stay open.
  export interface Driver {
    [group: string]: Record<string, (event?: any) => any>
  }
  // what an event answers: its shape is the platform's and stays open
  export type Reply = any
  export type On = (event: string, handler: (ctx: any, event: any) => unknown) => void

  export function describe(name: string, body: () => void): void
  export function test(name: string, body: ($: any, on: any) => unknown): void
  export function expect(value: unknown, message?: string): Matchers
  export const mock: {
    clock(on: On): MockClock
    env(on: On, variables: Record<string, string>): void
  }
}
