import { randomInt as cryptoRandomInt } from 'node:crypto';
import type { Credentials } from './clients';

export type SeededRole = 'user' | 'admin';

function required(
  env: Readonly<Record<string, string | undefined>>,
  key: string,
): string {
  const value = env[key];
  if (!value) {
    throw new Error(`${key} is not set`);
  }
  return value;
}

export function seededCredentials(
  env: Readonly<Record<string, string | undefined>>,
  role: SeededRole,
): Credentials {
  const prefix = role === 'admin' ? 'E2E_ADMIN' : 'E2E_USER';
  return {
    emailAddress: required(env, `${prefix}_EMAIL`),
    password: required(env, `${prefix}_PASSWORD`),
  };
}

export function newRunId(
  now: number = Date.now(),
  random: () => number = Math.random,
): string {
  return `${now.toString(36)}${Math.floor(random() * 36 ** 4).toString(36)}`;
}

// The sweeper keys on the e2e- prefix; the worker index keeps parallel
// workers in one run from registering the same address.
export function ephemeralEmail(
  runId: string,
  workerIndex: number,
  n: number,
): string {
  return `e2e-${runId}-w${workerIndex}-${n}@example.com`;
}

const CLASSES = [
  'abcdefghijkmnopqrstuvwxyz',
  'ABCDEFGHJKLMNPQRSTUVWXYZ',
  '23456789',
  '!@#$%^&*',
];

export function generatePassword(
  randomInt: (max: number) => number = cryptoRandomInt,
): string {
  const all = CLASSES.join('');
  const chars = CLASSES.map((set) => set[randomInt(set.length)]);
  while (chars.length < 16) chars.push(all[randomInt(all.length)]);
  for (let i = chars.length - 1; i > 0; i--) {
    const j = randomInt(i + 1);
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  return chars.join('');
}

export function isGoneOrDeleted(status: number): boolean {
  return status === 204 || status === 404;
}

export interface CleanupStatuses {
  readonly profile: number;
  readonly user: number;
}

export type Relogin =
  | { readonly kind: 'rejected'; readonly status: number }
  | { readonly kind: 'ok'; readonly retried: CleanupStatuses };

// The JVM identity service answers 401 to a token whose user is already
// deleted, where the Go service answers 204. A 401 is therefore ambiguous:
// only a fresh login tells "user is gone" from "token is bad".
export async function cleanupLeaks(
  initial: CleanupStatuses,
  relogin: () => Promise<Relogin>,
): Promise<string[]> {
  const entries = [
    ['profile', initial.profile],
    ['user', initial.user],
  ] as const;
  const leaks: string[] = [];
  const unauthorized: (typeof entries)[number][0][] = [];
  for (const [what, status] of entries) {
    if (status === 401) unauthorized.push(what);
    else if (!isGoneOrDeleted(status)) leaks.push(`${what}: HTTP ${status}`);
  }
  if (unauthorized.length === 0) return leaks;
  const outcome = await relogin();
  if (outcome.kind === 'rejected') {
    if (outcome.status !== 401) {
      leaks.push(`login: HTTP ${outcome.status}`);
    }
    return leaks;
  }
  for (const what of unauthorized) {
    const status = outcome.retried[what];
    if (!isGoneOrDeleted(status)) leaks.push(`${what}: HTTP ${status}`);
  }
  return leaks;
}
