import { describe, expect, it } from 'vitest';
import { createApiClients, login } from './clients';

function recordingFetch(reply: () => Response) {
  const seen: Request[] = [];
  const fetch = async (request: Request) => {
    seen.push(request);
    return reply();
  };
  return { fetch, seen };
}

describe('createApiClients', () => {
  it('attaches the bearer token to every call', async () => {
    const { fetch, seen } = recordingFetch(
      () =>
        new Response('[]', {
          status: 200,
          headers: { 'content-type': 'application/json' },
        }),
    );
    const { identity, profile } = createApiClients(fetch, 'https://h', 'tok');
    await identity.GET('/api/users');
    await profile.GET('/api/profiles');
    expect(seen.map((r) => r.headers.get('authorization'))).toEqual([
      'Bearer tok',
      'Bearer tok',
    ]);
    expect(seen[0].url).toBe('https://h/api/users');
  });

  it('sends no Authorization header without a token', async () => {
    const { fetch, seen } = recordingFetch(
      () => new Response('{}', { status: 200 }),
    );
    await createApiClients(fetch, 'https://h').identity.GET('/api/roles');
    expect(seen[0].headers.has('authorization')).toBe(false);
  });
});

describe('login', () => {
  it('returns the minted token', async () => {
    const { fetch } = recordingFetch(
      () =>
        new Response('{"jwtToken":"abc"}', {
          status: 200,
          headers: { 'content-type': 'application/json' },
        }),
    );
    const { identity } = createApiClients(fetch, 'https://h');
    await expect(
      login(identity, { emailAddress: 'a@example.com', password: 'p' }),
    ).resolves.toBe('abc');
  });

  it('fails with the status and never the password', async () => {
    const { fetch } = recordingFetch(() => new Response(null, { status: 401 }));
    const { identity } = createApiClients(fetch, 'https://h');
    const failure = login(identity, {
      emailAddress: 'a@example.com',
      password: 'Sup3r$ecret',
    });
    await expect(failure).rejects.toThrow(
      /Login for a@example.com failed: HTTP 401/,
    );
    await expect(failure).rejects.not.toThrow(/Sup3r\$ecret/);
  });
});

describe('typed calls', () => {
  it('accepts authorised operations without an explicit Authorization header', () => {
    // Compile-time check: a contract regeneration that makes the header
    // required again fails tsc here.
    const { identity, profile } = createApiClients(
      async () => new Response(null, { status: 204 }),
      'https://h',
    );
    const calls = [
      () =>
        identity.DELETE('/api/users/{userId}', {
          params: { path: { userId: 1 } },
        }),
      () =>
        profile.DELETE('/api/profiles/by-user/{userId}', {
          params: { path: { userId: 1 } },
        }),
    ];
    expect(calls).toHaveLength(2);
  });
});
