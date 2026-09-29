import type { Profile } from './profiles';

export const GATE = '@gate';
export const PRD_SAFE = '@prd-safe';
export const QUARANTINE = '@quarantine';
export const ADMIN = '@admin';

// The config's grep already narrows prd runs to prd-safe tests. This check
// runs inside every test before its first request so that a config edit, a new
// project or a dropped filter still cannot put a mutating test in front of prd.
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
