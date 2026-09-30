import { describe, expect, it } from 'vitest';
import {
  cleanupLeaks,
  ephemeralEmail,
  generatePassword,
  isGoneOrDeleted,
  newRunId,
  seededCredentials,
} from './test-users';

const EMAIL_PATTERN = /^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$/;

describe('seededCredentials', () => {
  it('reads the user pair', () => {
    expect(
      seededCredentials(
        { E2E_USER_EMAIL: 'u@example.com', E2E_USER_PASSWORD: 'p' },
        'user',
      ),
    ).toEqual({
      emailAddress: 'u@example.com',
      password: 'p',
    });
  });

  it('names the missing variable instead of letting the API answer 401', () => {
    expect(() =>
      seededCredentials({ E2E_ADMIN_EMAIL: 'a@example.com' }, 'admin'),
    ).toThrow(/E2E_ADMIN_PASSWORD is not set/);
  });
});

describe('ephemeral identities', () => {
  it('builds run ids of lowercase base36', () => {
    expect(newRunId(0, () => 0.5)).toMatch(/^[0-9a-z]+$/);
  });

  it('builds addresses the contract accepts and workers cannot collide on', () => {
    const a = ephemeralEmail('r1', 0, 1);
    const b = ephemeralEmail('r1', 1, 1);
    expect(a).toBe('e2e-r1-w0-1@example.com');
    expect(a).not.toBe(b);
    expect(a).toMatch(EMAIL_PATTERN);
  });

  it('generates passwords meeting every class the contract requires', () => {
    for (let i = 0; i < 50; i++) {
      const password = generatePassword();
      expect(password.length).toBeGreaterThanOrEqual(16);
      expect(password).toMatch(/[a-z]/);
      expect(password).toMatch(/[A-Z]/);
      expect(password).toMatch(/[0-9]/);
      expect(password).toMatch(/[^a-zA-Z0-9]/);
    }
  });
});

describe('isGoneOrDeleted', () => {
  it('treats deleted and already-gone as clean, anything else as a leak', () => {
    expect(isGoneOrDeleted(204)).toBe(true);
    expect(isGoneOrDeleted(404)).toBe(true);
    expect(isGoneOrDeleted(401)).toBe(false);
    expect(isGoneOrDeleted(500)).toBe(false);
  });
});

describe('cleanupLeaks', () => {
  const neverRelogin = () => {
    throw new Error('relogin must not run');
  };
  const rejected = (status: number) => async () =>
    ({ kind: 'rejected', status }) as const;
  const retried = (profile: number, user: number) => async () =>
    ({ kind: 'ok', retried: { profile, user } }) as const;

  it('treats 204 and 404 as clean without logging in again', async () => {
    expect(
      await cleanupLeaks({ profile: 204, user: 404 }, neverRelogin),
    ).toEqual([]);
  });

  it('treats 401 followed by a rejected login as already gone', async () => {
    expect(
      await cleanupLeaks({ profile: 404, user: 401 }, rejected(401)),
    ).toEqual([]);
  });

  it('is clean when a fresh token retries to 204', async () => {
    expect(
      await cleanupLeaks({ profile: 401, user: 401 }, retried(204, 204)),
    ).toEqual([]);
  });

  it.each([401, 500])('leaks when the retry answers %i', async (status) => {
    expect(
      await cleanupLeaks({ profile: 204, user: 401 }, retried(204, status)),
    ).toEqual([`user: HTTP ${status}`]);
  });

  it('leaks a 500 without logging in again', async () => {
    expect(
      await cleanupLeaks({ profile: 204, user: 500 }, neverRelogin),
    ).toEqual(['user: HTTP 500']);
  });

  it('leaks when the login fails with anything but 401', async () => {
    expect(
      await cleanupLeaks({ profile: 401, user: 204 }, rejected(503)),
    ).toEqual(['login: HTTP 503']);
  });
});
