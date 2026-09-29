import { describe, expect, it } from 'vitest';
import {
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
