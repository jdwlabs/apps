import createClient, { type Client } from 'openapi-fetch';
import type { paths as IdentityPaths } from './generated/identity';
import type { paths as ProfilePaths } from './generated/profile';
import type { FetchLike } from './request-fetch';

export type IdentityClient = Client<IdentityPaths>;
export type ProfileClient = Client<ProfilePaths>;

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
  const identity = createClient<IdentityPaths>({ baseUrl, fetch });
  const profile = createClient<ProfilePaths>({ baseUrl, fetch });
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
