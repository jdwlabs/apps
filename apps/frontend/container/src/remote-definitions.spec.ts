import {
  RemoteDefinitions,
  resolveRemoteDefinitions,
} from './remote-definitions';

const fallback: RemoteDefinitions = {
  authui: 'http://localhost:4201',
  usersui: 'http://localhost:4202',
};

describe('resolveRemoteDefinitions', () => {
  it('keeps a fully valid payload', () => {
    const payload = {
      authui: 'https://authui.non.jdwlabs.com',
      usersui: 'https://usersui.non.jdwlabs.com',
    };

    expect(resolveRemoteDefinitions(payload, fallback)).toEqual({
      definitions: payload,
      usedFallback: false,
      ignored: [],
    });
  });

  it('falls back when the payload is null', () => {
    expect(resolveRemoteDefinitions(null, fallback)).toEqual({
      definitions: fallback,
      usedFallback: true,
      ignored: [],
    });
  });

  it('falls back when the payload is an empty object', () => {
    expect(resolveRemoteDefinitions({}, fallback)).toEqual({
      definitions: fallback,
      usedFallback: true,
      ignored: [],
    });
  });

  it.each([
    ['undefined', undefined],
    ['a string', 'authui'],
    ['a number', 42],
    ['an array', ['authui']],
  ])('falls back when the payload is %s', (_label, payload) => {
    expect(resolveRemoteDefinitions(payload, fallback)).toEqual({
      definitions: fallback,
      usedFallback: true,
      ignored: [],
    });
  });

  it('falls back when every entry is unusable, reporting each name', () => {
    const payload = { authui: null, usersui: '', rolesui: '   ' };

    expect(resolveRemoteDefinitions(payload, fallback)).toEqual({
      definitions: fallback,
      usedFallback: true,
      ignored: ['authui', 'usersui', 'rolesui'],
    });
  });

  it('keeps usable entries and reports the ones it drops', () => {
    const payload = {
      authui: 'https://authui.non.jdwlabs.com',
      usersui: null,
      rolesui: 7,
    };

    expect(resolveRemoteDefinitions(payload, fallback)).toEqual({
      definitions: { authui: 'https://authui.non.jdwlabs.com' },
      usedFallback: false,
      ignored: ['usersui', 'rolesui'],
    });
  });

  it('does not mutate the fallback when the payload is unusable', () => {
    resolveRemoteDefinitions({ authui: null }, fallback);

    expect(fallback).toEqual({
      authui: 'http://localhost:4201',
      usersui: 'http://localhost:4202',
    });
  });
});
