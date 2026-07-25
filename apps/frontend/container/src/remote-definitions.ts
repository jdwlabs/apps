export type RemoteDefinitions = Record<string, string>;

export interface ResolvedRemoteDefinitions {
  definitions: RemoteDefinitions;
  usedFallback: boolean;
  ignored: string[];
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isRemoteUrl(value: unknown): value is string {
  return typeof value === 'string' && value.trim() !== '';
}

/**
 * Service discovery answers `200 application/json` with a `null` body when an
 * environment has no remotes configured, so a bare promise chain treats an
 * unusable payload as success and hands it straight to module federation.
 * Anything that is not a usable definition set resolves to the fallback here
 * instead, keeping the shell renderable when discovery is empty or malformed.
 */
export function resolveRemoteDefinitions(
  payload: unknown,
  fallback: RemoteDefinitions,
): ResolvedRemoteDefinitions {
  if (!isPlainObject(payload)) {
    return { definitions: fallback, usedFallback: true, ignored: [] };
  }

  const entries = Object.entries(payload);
  const usable = entries.filter(([, url]) => isRemoteUrl(url));
  const ignored = entries
    .filter(([, url]) => !isRemoteUrl(url))
    .map(([name]) => name);

  if (usable.length === 0) {
    return { definitions: fallback, usedFallback: true, ignored };
  }

  return {
    definitions: Object.fromEntries(usable) as RemoteDefinitions,
    usedFallback: false,
    ignored,
  };
}
