import type { APIRequestContext } from '@playwright/test';

export type FetchLike = (input: Request) => Promise<Response>;

// Routing openapi-fetch through Playwright's request context, rather than
// Node's fetch, is what puts every call in the Playwright trace and lets the
// context carry baseURL, Host and TLS settings from the profile.
export function requestFetch(request: APIRequestContext): FetchLike {
  return async (input) => {
    const raw = Buffer.from(await input.arrayBuffer());
    // A redirect from the API is itself a finding (an HTTP listener bouncing
    // to HTTPS, a gateway misroute); following it would hide it.
    const response = await request.fetch(input.url, {
      method: input.method,
      headers: Object.fromEntries(input.headers.entries()),
      data: raw.length > 0 ? raw : undefined,
      failOnStatusCode: false,
      maxRedirects: 0,
    });
    const status = response.status();
    const hasNoBody = status === 204 || status === 205 || status === 304;
    return new Response(
      hasNoBody ? null : new Uint8Array(await response.body()),
      {
        status,
        statusText: response.statusText(),
        headers: response
          .headersArray()
          .map(({ name, value }) => [name, value] as [string, string]),
      },
    );
  };
}
