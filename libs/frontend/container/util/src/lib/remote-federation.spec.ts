import { remoteEntryUrl } from './remote-federation';

describe('remoteEntryUrl', () => {
  it.each([
    ['http://localhost:4201', 'http://localhost:4201/remoteEntry.mjs'],
    ['http://localhost:4201/', 'http://localhost:4201/remoteEntry.mjs'],
    [
      'https://cdn.example.com/auth',
      'https://cdn.example.com/auth/remoteEntry.mjs',
    ],
    [
      'http://localhost:4201/remoteEntry.mjs',
      'http://localhost:4201/remoteEntry.mjs',
    ],
    [
      'http://localhost:4201/mf-manifest.json',
      'http://localhost:4201/mf-manifest.json',
    ],
    [
      'http://localhost:4201/remoteEntry.js?v=2',
      'http://localhost:4201/remoteEntry.js?v=2',
    ],
  ])('maps %s to %s', (input, expected) => {
    expect(remoteEntryUrl(input)).toBe(expected);
  });
});
