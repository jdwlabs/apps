import {
  loadRemote,
  registerRemotes,
} from '@module-federation/enhanced/runtime';

/**
 * Service discovery hands out either a remote's base URL or the full entry
 * URL. The runtime needs the entry file, so a base URL is completed here, with
 * any query string or fragment kept after the file name. Relative definitions
 * resolve against the page origin, as `@nx/angular/mf` did.
 */
export function remoteEntryUrl(
  url: string,
  origin: string = globalThis.location?.origin ?? 'http://localhost',
): string {
  const entry = new URL(url.trim(), new URL(origin).origin);
  if (!/\.(mjs|js|json)$/.test(entry.pathname)) {
    entry.pathname = `${entry.pathname.replace(/\/+$/, '')}/remoteEntry.mjs`;
  }
  return entry.href;
}

/** Container names are normalised to a valid identifier when the host builds. */
function containerName(remoteName: string): string {
  return remoteName
    .replace(/^[^a-zA-Z_$]/, '_')
    .replace(/[^a-zA-Z0-9_$]/g, '_');
}

export function setRemoteDefinitions(
  definitions: Record<string, string>,
): void {
  registerRemotes(
    Object.entries(definitions).map(([name, url]) => {
      const entry = remoteEntryUrl(url);
      return {
        name: containerName(name),
        entry,
        // A manifest carries its own type; a bare entry file is an ES module.
        ...(new URL(entry).pathname.endsWith('.json')
          ? {}
          : { type: 'module' as const }),
      };
    }),
    { force: true },
  );
}

export async function loadRemoteModule<T = Record<string, unknown>>(
  remoteName: string,
  moduleName: string,
): Promise<T> {
  const id = `${containerName(remoteName)}/${moduleName.replace(/^\.\//, '')}`;
  const loaded = await loadRemote<T>(id);
  if (!loaded) {
    throw new Error(`Remote module ${id} could not be loaded`);
  }
  return loaded;
}
