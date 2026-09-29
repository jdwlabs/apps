import { describe, expect, it } from 'vitest';
import {
  versionFromInfo,
  waitForVersions,
  type WaitDeps,
} from './version-wait';

function clock(observations: Record<string, (string | undefined)[]>): WaitDeps {
  let t = 0;
  const cursor: Record<string, number> = {};
  return {
    now: () => t,
    sleep: async (ms) => {
      t += ms;
    },
    fetchVersion: async (url) => {
      const seen = observations[url];
      const i = Math.min(cursor[url] ?? 0, seen.length - 1);
      cursor[url] = i + 1;
      return seen[i];
    },
  };
}

const targets = [
  { service: 'identity-service', infoUrl: 'i', expected: '1.4.0' },
  { service: 'profile-service', infoUrl: 'p', expected: '0.9.0' },
];

describe('waitForVersions', () => {
  it('returns once every service serves its expected version', async () => {
    const deps = clock({ i: ['1.3.0', '1.4.0'], p: [undefined, '0.9.0'] });
    await expect(
      waitForVersions(targets, deps, { timeoutMs: 60_000, intervalMs: 5_000 }),
    ).resolves.toBeUndefined();
  });

  it('times out naming each lagging service and what it last served', async () => {
    const deps = clock({ i: ['1.4.0'], p: ['0.8.0'] });
    await expect(
      waitForVersions(targets, deps, { timeoutMs: 20_000, intervalMs: 5_000 }),
    ).rejects.toThrow(/profile-service: expected 0\.9\.0, last saw 0\.8\.0/);
  });

  it('reports an unreachable service as such', async () => {
    const deps = clock({ i: ['1.4.0'], p: [undefined] });
    await expect(
      waitForVersions(targets, deps, { timeoutMs: 10_000, intervalMs: 5_000 }),
    ).rejects.toThrow(/profile-service: expected 0\.9\.0, last saw nothing/);
  });
});

describe('versionFromInfo', () => {
  it('reads the actuator build-info shape', () => {
    expect(versionFromInfo({ build: { version: '1.4.0' } })).toBe('1.4.0');
    expect(versionFromInfo({})).toBeUndefined();
    expect(versionFromInfo('nope')).toBeUndefined();
  });
});
