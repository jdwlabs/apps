import { test as base, type APIRequestContext } from '@playwright/test';
import {
  createApiClients,
  login,
  type ApiClients,
  type Credentials,
} from './clients';
import { resolveProfile, type Profile } from './profiles';
import { requestFetch } from './request-fetch';
import { assertAllowed } from './tags';
import {
  ephemeralEmail,
  cleanupLeaks,
  generatePassword,
  newRunId,
  seededCredentials,
} from './test-users';

export interface EphemeralUser {
  readonly id: number;
  readonly credentials: Credentials;
  readonly clients: ApiClients;
}

interface TestFixtures {
  guard: void;
  apiRequest: APIRequestContext;
  anonymous: ApiClients;
  asUser: ApiClients;
  asAdmin: ApiClients;
  newEphemeralUser: () => Promise<EphemeralUser>;
}

interface WorkerFixtures {
  profile: Profile;
  runId: string;
}

export const test = base.extend<TestFixtures, WorkerFixtures>({
  profile: [
    // Playwright requires the first fixture argument to be an object pattern.
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => use(resolveProfile(process.env)),
    { scope: 'worker' },
  ],
  runId: [
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => use(process.env['E2E_RUN_ID'] ?? newRunId()),
    { scope: 'worker' },
  ],

  // Covers test-scoped fixtures only. A future worker-scoped fixture or
  // beforeAll hook that makes requests runs outside it and must call
  // assertAllowed itself.
  guard: [
    async ({ profile }, use, testInfo) => {
      assertAllowed(profile, testInfo.tags, testInfo.title);
      await use();
    },
    { auto: true },
  ],

  // Depends on guard so no request context exists before the tag check passes.
  apiRequest: async ({ playwright, profile, guard }, use) => {
    void guard;
    const context = await playwright.request.newContext({
      baseURL: profile.apiBaseUrl,
      ignoreHTTPSErrors: profile.ignoreHttpsErrors,
      extraHTTPHeaders: profile.hostHeader ? { Host: profile.hostHeader } : {},
    });
    await use(context);
    await context.dispose();
  },

  anonymous: async ({ apiRequest, profile }, use) => {
    await use(createApiClients(requestFetch(apiRequest), profile.apiBaseUrl));
  },

  asUser: async ({ apiRequest, profile, anonymous }, use) => {
    const token = await login(
      anonymous.identity,
      seededCredentials(process.env, 'user'),
    );
    await use(
      createApiClients(requestFetch(apiRequest), profile.apiBaseUrl, token),
    );
  },

  asAdmin: async ({ apiRequest, profile, anonymous }, use) => {
    if (profile.prdSafeOnly) {
      throw new Error(`The seeded admin is never used on ${profile.name}`);
    }
    const token = await login(
      anonymous.identity,
      seededCredentials(process.env, 'admin'),
    );
    await use(
      createApiClients(requestFetch(apiRequest), profile.apiBaseUrl, token),
    );
  },

  newEphemeralUser: async (
    { apiRequest, profile, anonymous, runId },
    use,
    testInfo,
  ) => {
    if (!profile.allowsEphemeralUsers) {
      throw new Error(`Ephemeral users are never created on ${profile.name}`);
    }
    const pending: {
      id: number;
      credentials: Credentials;
      user?: EphemeralUser;
    }[] = [];
    let n = 0;
    await use(async () => {
      n += 1;
      const credentials = {
        emailAddress: ephemeralEmail(runId, testInfo.workerIndex, n),
        password: generatePassword(),
      };
      const { data, response } = await anonymous.identity.POST('/auth/user', {
        body: credentials,
      });
      if (!data?.id) {
        throw new Error(
          `Registering ${credentials.emailAddress} failed: HTTP ${response.status}`,
        );
      }
      // Tracked before login so a failed login still gets the user cleaned up.
      const entry: (typeof pending)[number] = {
        id: data.id,
        credentials,
      };
      pending.push(entry);
      const token = await login(anonymous.identity, credentials);
      const user = {
        id: data.id,
        credentials,
        clients: createApiClients(
          requestFetch(apiRequest),
          profile.apiBaseUrl,
          token,
        ),
      };
      entry.user = user;
      return user;
    });
    // Teardown runs even when the test failed, so a broken gate cannot leave
    // users behind. Profile first is belt-and-braces: authorisation compares
    // the token's stateless user id claim, so either order works today, but
    // this one never depends on a user delete leaving the profile reachable.
    // A test may delete its own user; the JVM identity service then answers
    // 401 to that user's token where the Go service answers 204, so a 401 is
    // only a leak if a fresh login still succeeds.
    const leaks: string[] = [];
    const deleteBoth = async (clients: ApiClients, id: number) => ({
      profile: (
        await clients.profile.DELETE('/api/profiles/by-user/{userId}', {
          params: { path: { userId: id } },
        })
      ).response.status,
      user: (
        await clients.identity.DELETE('/api/users/{userId}', {
          params: { path: { userId: id } },
        })
      ).response.status,
    });
    for (const { id, credentials, user } of pending) {
      try {
        const clients =
          user?.clients ??
          createApiClients(
            requestFetch(apiRequest),
            profile.apiBaseUrl,
            await login(anonymous.identity, credentials),
          );
        const found = await cleanupLeaks(
          await deleteBoth(clients, id),
          async () => {
            const { data, response } = await anonymous.identity.POST(
              '/auth/authenticate',
              { body: credentials },
            );
            if (!data) return { kind: 'rejected', status: response.status };
            const fresh = createApiClients(
              requestFetch(apiRequest),
              profile.apiBaseUrl,
              data.jwtToken,
            );
            return { kind: 'ok', retried: await deleteBoth(fresh, id) };
          },
        );
        for (const leak of found) {
          leaks.push(`${credentials.emailAddress} ${leak}`);
        }
      } catch (error) {
        leaks.push(
          `${credentials.emailAddress}: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
    }
    if (leaks.length > 0) {
      throw new Error(
        `Ephemeral cleanup left resources behind:\n  ${leaks.join('\n  ')}`,
      );
    }
  },
});

export { expect } from '@playwright/test';
