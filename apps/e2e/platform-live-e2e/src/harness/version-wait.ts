export interface VersionTarget {
  readonly service: string;
  readonly infoUrl: string;
  readonly expected: string;
}

export interface WaitDeps {
  fetchVersion(infoUrl: string): Promise<string | undefined>;
  sleep(ms: number): Promise<void>;
  now(): number;
}

// Testing before ArgoCD finishes rolling would grade the previous pods, so the
// gate waits until every service reports the version it was triggered for.
export async function waitForVersions(
  targets: readonly VersionTarget[],
  deps: WaitDeps,
  options: { timeoutMs: number; intervalMs: number },
): Promise<void> {
  const deadline = deps.now() + options.timeoutMs;
  const lastSeen = new Map<string, string | undefined>();
  for (;;) {
    const observed = await Promise.all(
      targets.map((t) => deps.fetchVersion(t.infoUrl).catch(() => undefined)),
    );
    targets.forEach((t, i) => lastSeen.set(t.service, observed[i]));
    const lagging = targets.filter(
      (t) => lastSeen.get(t.service) !== t.expected,
    );
    if (lagging.length === 0) return;
    if (deps.now() >= deadline) {
      const detail = lagging.map(
        (t) =>
          `${t.service}: expected ${t.expected}, last saw ${lastSeen.get(t.service) ?? 'nothing'}`,
      );
      throw new Error(
        `Timed out after ${options.timeoutMs}ms waiting for rollout:\n  ${detail.join('\n  ')}`,
      );
    }
    await deps.sleep(options.intervalMs);
  }
}

export function versionFromInfo(body: unknown): string | undefined {
  if (typeof body !== 'object' || body === null) return undefined;
  const build = (body as { build?: unknown }).build;
  if (typeof build !== 'object' || build === null) return undefined;
  const version = (build as { version?: unknown }).version;
  return typeof version === 'string' ? version : undefined;
}
