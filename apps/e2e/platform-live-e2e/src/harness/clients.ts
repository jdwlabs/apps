import createClient, { type Client } from 'openapi-fetch';
import type { paths as IdentityPaths } from './generated/identity';
import type { paths as ProfilePaths } from './generated/profile';
import type { FetchLike } from './request-fetch';

type OptionalAuthorization<Params> = Params extends {
  header: infer H;
}
  ? H extends { Authorization: string }
    ? Omit<Params, 'header'> & {
        header?: Omit<H, 'Authorization'> & { Authorization?: string };
      }
    : Params
  : Params;

// The contracts declare Authorization as a required header parameter, but the
// bearer middleware supplies it, so call sites must not have to repeat it.
type BearerSupplied<P> = {
  [Path in keyof P]: {
    [Method in keyof P[Path]]: P[Path][Method] extends {
      parameters: infer Params;
    }
      ? Omit<P[Path][Method], 'parameters'> & {
          parameters: OptionalAuthorization<Params>;
        }
      : P[Path][Method];
  };
};

export type IdentityClient = Client<BearerSupplied<IdentityPaths>>;
export type ProfileClient = Client<BearerSupplied<ProfilePaths>>;

export interface ApiClients {
  readonly identity: IdentityClient;
  readonly profile: ProfileClient;
}

export interface Credentials {
  readonly emailAddress: string;
  readonly password: string;
}

export function createApiClients(
  fetch: FetchLike,
  baseUrl: string,
  token?: string,
): ApiClients {
  const identity = createClient<BearerSupplied<IdentityPaths>>({
    baseUrl,
    fetch,
  });
  const profile = createClient<BearerSupplied<ProfilePaths>>({
    baseUrl,
    fetch,
  });
  if (token) {
    const bearer = {
      onRequest({ request }: { request: Request }) {
        request.headers.set('Authorization', `Bearer ${token}`);
        return request;
      },
    };
    identity.use(bearer);
    profile.use(bearer);
  }
  return { identity, profile };
}

export async function login(
  identity: IdentityClient,
  credentials: Credentials,
): Promise<string> {
  const { data, response } = await identity.POST('/auth/authenticate', {
    body: credentials,
  });
  if (!data) {
    throw new Error(
      `Login for ${credentials.emailAddress} failed: HTTP ${response.status}`,
    );
  }
  return data.jwtToken;
}
