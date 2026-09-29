export type ProfileName =
  'non-public' | 'prd-public' | 'non-incluster' | 'prd-incluster';

export interface Profile {
  readonly name: ProfileName;
  readonly environment: 'non' | 'prd';
  readonly apiBaseUrl: string;
  readonly hostHeader: string | undefined;
  readonly ignoreHttpsErrors: boolean;
  readonly prdSafeOnly: boolean;
  readonly allowsEphemeralUsers: boolean;
}

// Pods cannot reach the public hostnames because the router does not hairpin,
// so in-cluster runs dial the gateway Service and name the public host
// themselves. The gateway's certificate is for the public host, not the
// Service name, hence ignoreHttpsErrors on those profiles only.
const GATEWAY = 'https://platform-gateway-nginx.nginx-gateway.svc:443';

function define(
  name: ProfileName,
  environment: 'non' | 'prd',
  inCluster: boolean,
): Profile {
  const publicHost = `usersrole.${environment}.jdwlabs.com`;
  return {
    name,
    environment,
    apiBaseUrl: inCluster ? GATEWAY : `https://${publicHost}`,
    hostHeader: inCluster ? publicHost : undefined,
    ignoreHttpsErrors: inCluster,
    prdSafeOnly: environment === 'prd',
    allowsEphemeralUsers: environment !== 'prd',
  };
}

const PROFILES: Record<ProfileName, Profile> = {
  'non-public': define('non-public', 'non', false),
  'prd-public': define('prd-public', 'prd', false),
  'non-incluster': define('non-incluster', 'non', true),
  'prd-incluster': define('prd-incluster', 'prd', true),
};

export const PROFILE_NAMES = Object.keys(PROFILES) as readonly ProfileName[];

export function profileByName(name: ProfileName): Profile {
  return PROFILES[name];
}

export function resolveProfile(
  env: Readonly<Record<string, string | undefined>>,
): Profile {
  const requested = env['E2E_PROFILE'];
  const valid = PROFILE_NAMES.join(', ');
  if (!requested) {
    throw new Error(`E2E_PROFILE is required; one of: ${valid}`);
  }
  if (!(requested in PROFILES)) {
    throw new Error(`Unknown E2E_PROFILE "${requested}"; one of: ${valid}`);
  }
  return PROFILES[requested as ProfileName];
}
