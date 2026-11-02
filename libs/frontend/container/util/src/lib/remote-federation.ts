import {
  loadRemote,
  registerRemotes,
} from '@module-federation/enhanced/runtime';

/**
 * Service discovery hands out either a remote's origin or the full entry URL.
 * The runtime needs the entry file, so a bare origin is completed here.
 */
export function remoteEntryUrl(url: string): string {
  const trimmed = url.trim();
  const { pathname } = new URL(trimmed, 'http://placeholder');
  return /\.(mjs|js|json)$/.test(pathname)
    ? trimmed
    : `${trimmed.replace(/\/+$/, '')}/remoteEntry.mjs`;
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
        ...(entry.endsWith('.json') ? {} : { type: 'module' as const }),
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
