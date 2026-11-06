import { remoteEntryUrl } from './remote-federation';

describe('remoteEntryUrl', () => {
  it.each([
    // Bare origins and trailing slashes.
    ['http://localhost:4201', 'http://localhost:4201/remoteEntry.mjs'],
    ['http://localhost:4201/', 'http://localhost:4201/remoteEntry.mjs'],
    ['http://localhost:4201//', 'http://localhost:4201/remoteEntry.mjs'],
    ['  http://localhost:4201  ', 'http://localhost:4201/remoteEntry.mjs'],
    [
      'https://cdn.example.com/auth',
      'https://cdn.example.com/auth/remoteEntry.mjs',
    ],
    [
      'https://cdn.example.com/auth/',
      'https://cdn.example.com/auth/remoteEntry.mjs',
    ],
    // Entry files and manifests are used as given.
    [
      'http://localhost:4201/remoteEntry.mjs',
      'http://localhost:4201/remoteEntry.mjs',
    ],
    [
      'http://localhost:4201/remoteEntry.js',
      'http://localhost:4201/remoteEntry.js',
    ],
    [
      'http://localhost:4201/mf-manifest.json',
      'http://localhost:4201/mf-manifest.json',
    ],
    // Query strings and fragments stay after the entry file.
    [
      'http://localhost:4201/remoteEntry.js?v=2',
      'http://localhost:4201/remoteEntry.js?v=2',
    ],
    ['http://localhost:4201?v=2', 'http://localhost:4201/remoteEntry.mjs?v=2'],
    [
      'https://cdn.example.com/auth/?v=2#top',
      'https://cdn.example.com/auth/remoteEntry.mjs?v=2#top',
    ],
    [
      'https://cdn.example.com/auth?token=a/b.js',
      'https://cdn.example.com/auth/remoteEntry.mjs?token=a/b.js',
    ],
  ])('maps %s to %s', (input, expected) => {
    expect(remoteEntryUrl(input)).toBe(expected);
  });

  it.each([
    ['/authui', 'https://app.example.com/authui/remoteEntry.mjs'],
    ['/authui/', 'https://app.example.com/authui/remoteEntry.mjs'],
    ['authui', 'https://app.example.com/authui/remoteEntry.mjs'],
    ['/authui?v=2', 'https://app.example.com/authui/remoteEntry.mjs?v=2'],
    [
      '/authui/remoteEntry.mjs',
      'https://app.example.com/authui/remoteEntry.mjs',
    ],
    ['/', 'https://app.example.com/remoteEntry.mjs'],
  ])('resolves relative %s against the origin', (input, expected) => {
    expect(remoteEntryUrl(input, 'https://app.example.com')).toBe(expected);
  });

  it('ignores the base path, resolving against the origin only', () => {
    expect(remoteEntryUrl('authui', 'https://app.example.com/shell/')).toBe(
      'https://app.example.com/authui/remoteEntry.mjs',
    );
  });

  it('defaults the base to the page origin', () => {
    expect(remoteEntryUrl('/authui')).toBe(
      `${globalThis.location.origin}/authui/remoteEntry.mjs`,
    );
  });
});
