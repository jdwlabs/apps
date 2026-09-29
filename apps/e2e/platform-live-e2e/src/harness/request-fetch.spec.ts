import { describe, expect, it } from 'vitest';
import type { APIRequestContext } from '@playwright/test';
import { requestFetch } from './request-fetch';

interface Call {
  url: string;
  options: Record<string, unknown>;
}

function fakeContext(reply: {
  status: number;
  headers?: Record<string, string>;
  body?: Buffer;
}) {
  const calls: Call[] = [];
  const context = {
    fetch: async (url: string, options: Record<string, unknown>) => {
      calls.push({ url, options });
      return {
        status: () => reply.status,
        statusText: () => '',
        headersArray: () =>
          Object.entries(reply.headers ?? {}).map(([name, value]) => ({
            name,
            value,
          })),
        body: async () => reply.body ?? Buffer.alloc(0),
      };
    },
  } as unknown as APIRequestContext;
  return { context, calls };
}

describe('requestFetch', () => {
  it('forwards method, url, headers and JSON body, and never throws on status', async () => {
    const { context, calls } = fakeContext({
      status: 201,
      headers: { 'content-type': 'application/json' },
      body: Buffer.from('{"id":7}'),
    });
    const response = await requestFetch(context)(
      new Request('https://h/api/roles', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: '{"name":"x"}',
      }),
    );
    expect(calls[0].url).toBe('https://h/api/roles');
    expect(calls[0].options).toMatchObject({
      method: 'POST',
      failOnStatusCode: false,
      maxRedirects: 0,
    });
    expect(
      (calls[0].options['headers'] as Record<string, string>)['content-type'],
    ).toBe('application/json');
    expect((calls[0].options['data'] as Buffer).toString()).toBe(
      '{"name":"x"}',
    );
    expect(response.status).toBe(201);
    expect(await response.json()).toEqual({ id: 7 });
  });

  it('sends no body on GET', async () => {
    const { context, calls } = fakeContext({ status: 200 });
    await requestFetch(context)(new Request('https://h/api/users'));
    expect(calls[0].options['data']).toBeUndefined();
  });

  it('returns a null body for 204 instead of throwing', async () => {
    const { context } = fakeContext({ status: 204 });
    const response = await requestFetch(context)(
      new Request('https://h/x', { method: 'DELETE' }),
    );
    expect(response.status).toBe(204);
    expect(response.body).toBeNull();
  });

  it('passes binary bodies through byte for byte', async () => {
    const png = Buffer.from([0x89, 0x50, 0x4e, 0x47]);
    const { context } = fakeContext({
      status: 200,
      headers: { 'content-type': 'image/png' },
      body: png,
    });
    const response = await requestFetch(context)(new Request('https://h/icon'));
    expect(Buffer.from(await response.arrayBuffer())).toEqual(png);
  });

  it('keeps the multipart boundary the Request encoded', async () => {
    const { context, calls } = fakeContext({ status: 200 });
    const form = new FormData();
    form.append(
      'icon',
      new Blob([Buffer.from([1, 2, 3])], { type: 'image/png' }),
      'icon.png',
    );
    await requestFetch(context)(
      new Request('https://h/icon', { method: 'POST', body: form }),
    );
    const headers = calls[0].options['headers'] as Record<string, string>;
    const boundary = /boundary=(.+)$/.exec(headers['content-type'])?.[1];
    expect(boundary).toBeTruthy();
    expect((calls[0].options['data'] as Buffer).toString('latin1')).toContain(
      `--${boundary}`,
    );
  });
});
