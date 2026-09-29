import { describe, expect, it } from 'vitest';
import { profileByName } from './profiles';
import { assertAllowed, gateFilter, GATE, PRD_SAFE } from './tags';

describe('assertAllowed', () => {
  it('lets any tagged test run on non', () => {
    expect(() =>
      assertAllowed(profileByName('non-public'), [GATE], 't'),
    ).not.toThrow();
  });

  it('refuses a test without @prd-safe on prd, whatever grep selected it', () => {
    for (const name of ['prd-public', 'prd-incluster'] as const) {
      expect(() =>
        assertAllowed(profileByName(name), [GATE], 'creates a role'),
      ).toThrow(
        /"creates a role" is not tagged @prd-safe and cannot run on prd-/,
      );
    }
  });

  it('allows @prd-safe tests on prd', () => {
    expect(() =>
      assertAllowed(profileByName('prd-public'), [GATE, PRD_SAFE], 't'),
    ).not.toThrow();
  });
});

describe('gateFilter', () => {
  it('selects the gate on non and only prd-safe tests on prd, never quarantined ones', () => {
    expect(gateFilter(profileByName('non-public'))).toEqual({
      grep: /@gate/,
      grepInvert: /@quarantine/,
    });
    expect(gateFilter(profileByName('prd-public'))).toEqual({
      grep: /@prd-safe/,
      grepInvert: /@quarantine/,
    });
  });
});
