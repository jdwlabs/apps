import { execFileSync } from 'node:child_process';
import { readCachedProjectGraph } from '@nx/devkit';
import DefaultChangelogRenderer from 'nx/release/changelog-renderer';
import TreeAwareChangelogRenderer from './changelog-renderer';

vi.mock('node:child_process', () => ({ execFileSync: vi.fn() }));
vi.mock('@nx/devkit', () => ({ readCachedProjectGraph: vi.fn() }));

const mockedExecFileSync = vi.mocked(execFileSync);
const mockedReadCachedProjectGraph = vi.mocked(readCachedProjectGraph);

const NO_CHANGES_TEXT =
  'This was a version bump only for usersrole to align it with other projects, there were no code changes.';

function makeRenderer(
  overrides: {
    project?: string | null;
    changes?: unknown[];
  } = {},
): TreeAwareChangelogRenderer {
  return new TreeAwareChangelogRenderer({
    changes: (overrides.changes ?? []) as never,
    changelogEntryVersion: '1.0.2',
    project: overrides.project === undefined ? 'usersrole' : overrides.project,
    entryWhenNoChanges: NO_CHANGES_TEXT,
    isVersionPlans: false,
    changelogRenderOptions: {},
    dependencyBumps: [],
    conventionalCommitsConfig: {} as never,
    remoteReleaseClient: {} as never,
  });
}

/** Routes execFileSync('git', [...]) calls by subcommand for a single test. */
function mockGit(responses: {
  describe?: string | Error;
  diff?: string | Error;
  log?: string | Error;
}) {
  mockedExecFileSync.mockImplementation((_cmd, args) => {
    const argv = args as string[];
    const subcommand = argv[0];
    const response = responses[subcommand as 'describe' | 'diff' | 'log'];
    if (response instanceof Error) {
      throw response;
    }
    if (response === undefined) {
      throw new Error(`unexpected git ${subcommand} invocation in test`);
    }
    return response;
  });
}

describe('TreeAwareChangelogRenderer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedReadCachedProjectGraph.mockReturnValue({
      nodes: {
        usersrole: { data: { root: 'apps/backend/usersrole' } },
      },
    } as never);
  });

  it('defers to the default renderer when there are bump-triggering changes', async () => {
    const renderSpy = vi
      .spyOn(DefaultChangelogRenderer.prototype, 'render')
      .mockResolvedValue('SENTINEL FROM BASE RENDERER');

    const renderer = makeRenderer({
      changes: [{ type: 'feat', affectedProjects: ['usersrole'] }],
    });
    await expect(renderer.render()).resolves.toBe(
      'SENTINEL FROM BASE RENDERER',
    );
    expect(renderSpy).toHaveBeenCalledTimes(1);
    // A real change to render means git is never consulted.
    expect(mockedExecFileSync).not.toHaveBeenCalled();
  });

  it('defers to the default renderer for the workspace-level changelog (project: null)', async () => {
    const renderSpy = vi
      .spyOn(DefaultChangelogRenderer.prototype, 'render')
      .mockResolvedValue('SENTINEL');

    const renderer = makeRenderer({ project: null });
    await expect(renderer.render()).resolves.toBe('SENTINEL');
    expect(renderSpy).toHaveBeenCalledTimes(1);
  });

  it('still emits the "no code changes" entry when the project tree genuinely did not change', async () => {
    mockGit({
      describe: 'usersrole-1.0.1\n',
      diff: '', // no files differ between the previous tag and HEAD
    });

    const renderer = makeRenderer();
    const result = await renderer.render();

    expect(result).toContain(NO_CHANGES_TEXT);
    expect(result).not.toContain('Other changes');
  });

  it('lists the hidden commits instead of claiming no code changes when the tree did change', async () => {
    mockGit({
      describe: 'usersrole-1.0.1\n',
      diff: 'apps/backend/usersrole/build.gradle.kts\n',
      log: 'f85673289 chore(deps): migrate usersrole to Spring Boot 4 + Gradle 9\n',
    });

    const renderer = makeRenderer();
    const result = await renderer.render();

    expect(result).not.toContain(NO_CHANGES_TEXT);
    expect(result).toContain('Other changes');
    expect(result).toContain(
      'f85673289 chore(deps): migrate usersrole to Spring Boot 4 + Gradle 9',
    );
  });

  it('falls back to the default empty entry when there is no previous release tag (first release)', async () => {
    mockGit({ describe: new Error('fatal: no tag found') });

    const renderer = makeRenderer();
    const result = await renderer.render();

    expect(result).toContain(NO_CHANGES_TEXT);
    expect(mockedExecFileSync).toHaveBeenCalledTimes(1); // describe only, no diff/log
  });

  it('falls back to the default empty entry when the project root cannot be resolved', async () => {
    mockedReadCachedProjectGraph.mockImplementation(() => {
      throw new Error('no cached project graph');
    });

    const renderer = makeRenderer();
    const result = await renderer.render();

    expect(result).toContain(NO_CHANGES_TEXT);
    expect(mockedExecFileSync).not.toHaveBeenCalled();
  });
});
