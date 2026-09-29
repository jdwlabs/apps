import type { Profile } from './profiles';

export const GATE = '@gate';
export const PRD_SAFE = '@prd-safe';
export const QUARANTINE = '@quarantine';
export const ADMIN = '@admin';

// The config's grep narrows prd runs to prd-safe tests, but a --grep on the
// command line replaces it. This check runs inside every test before its
// first request, so no filter can put a mutating test in front of prd.
export function assertAllowed(
  profile: Profile,
  tags: readonly string[],
  title: string,
): void {
  if (profile.prdSafeOnly && !tags.includes(PRD_SAFE)) {
    throw new Error(
      `"${title}" is not tagged ${PRD_SAFE} and cannot run on ${profile.name}`,
    );
  }
}

export function gateFilter(profile: Profile): {
  grep: RegExp;
  grepInvert: RegExp;
} {
  return {
    grep: profile.prdSafeOnly ? /@prd-safe/ : /@gate/,
    grepInvert: /@quarantine/,
  };
}
