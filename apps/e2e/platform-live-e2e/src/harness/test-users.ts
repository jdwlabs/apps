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
