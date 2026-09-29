import { describe, expect, it } from 'vitest';
import { PROFILE_NAMES, profileByName, resolveProfile } from './profiles';

describe('resolveProfile', () => {
  it('refuses to guess when E2E_PROFILE is unset', () => {
    expect(() => resolveProfile({})).toThrow(
      /E2E_PROFILE is required.*non-public, prd-public, non-incluster, prd-incluster/,
    );
  });

  it('refuses an unknown profile and lists the valid ones', () => {
    expect(() => resolveProfile({ E2E_PROFILE: 'prod' })).toThrow(
      /Unknown E2E_PROFILE "prod"/,
    );
  });

  it('resolves each known profile', () => {
    for (const name of PROFILE_NAMES) {
      expect(resolveProfile({ E2E_PROFILE: name }).name).toBe(name);
    }
  });
});

describe('profiles', () => {
  it('sends public profiles to the public API host', () => {
    expect(profileByName('non-public')).toMatchObject({
      apiBaseUrl: 'https://usersrole.non.jdwlabs.com',
      hostHeader: undefined,
      ignoreHttpsErrors: false,
    });
    expect(profileByName('prd-public').apiBaseUrl).toBe(
      'https://usersrole.prd.jdwlabs.com',
    );
  });

  it('sends in-cluster profiles to the gateway Service with the public host', () => {
    expect(profileByName('non-incluster')).toMatchObject({
      apiBaseUrl: 'https://platform-gateway-nginx.nginx-gateway.svc:443',
      hostHeader: 'usersrole.non.jdwlabs.com',
      ignoreHttpsErrors: true,
    });
  });

  it('restricts every prd profile to prd-safe tests and no user creation', () => {
    for (const name of ['prd-public', 'prd-incluster'] as const) {
      expect(profileByName(name)).toMatchObject({
        environment: 'prd',
        prdSafeOnly: true,
        allowsEphemeralUsers: false,
      });
    }
    for (const name of ['non-public', 'non-incluster'] as const) {
      expect(profileByName(name)).toMatchObject({
        environment: 'non',
        prdSafeOnly: false,
        allowsEphemeralUsers: true,
      });
    }
  });
});
