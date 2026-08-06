import { execFileSync } from 'node:child_process';
import { readCachedProjectGraph } from '@nx/devkit';
import * as changelogRendererModule from 'nx/release/changelog-renderer';

/**
 * nx.json points nx release at this file by path, and nx loads it through
 * two different pipelines depending on context: vitest (esbuild) unwraps
 * the CJS `exports.default` down to the class in one step, but nx's own
 * release runtime loads it under Node's native TypeScript type-stripping,
 * which wraps a CJS module's own `exports.default` in an *additional*
 * synthetic default layer -- `changelogRendererModule.default` there is
 * `{ default: Class }`, not `Class` itself, so a plain `.default` access
 * (or a bare `import X from '...'`, which hits the same interop path)
 * throws `class Y extends X` at runtime. Unwrapping until we hit something
 * callable is what actually works under both loaders.
 */
function unwrapDefaultExport<T>(mod: unknown): T {
  let candidate = mod;
  while (
    candidate &&
    typeof candidate !== 'function' &&
    typeof (candidate as { default?: unknown }).default !== 'undefined'
  ) {
    candidate = (candidate as { default: unknown }).default;
  }
  return candidate as T;
}

const DefaultChangelogRenderer = unwrapDefaultExport<
  (typeof changelogRendererModule)['default']
>(changelogRendererModule);

/**
 * Nx filters conventional-commit types marked `changelog.hidden` (chore,
 * docs, build, ci, ...) out of the change list before a renderer ever sees
 * it. When every commit touching a project happens to be a hidden type, the
 * base renderer's `shouldRenderEmptyEntry` sees zero relevant changes and
 * falls back to `entryWhenNoChanges` -- "there were no code changes" -- even
 * though the project's tree genuinely moved (e.g. a `chore(deps)` migration).
 * That is what happened to usersrole 1.0.2: a Spring Boot 4 + Gradle 9
 * migration shipped under a changelog entry that denied any change occurred.
 *
 * This renderer only changes behaviour for that exact false-negative case:
 * before emitting the empty-entry text, it confirms with git that the
 * project's own path truly didn't change since its previous release tag. If
 * it did change, it lists the hidden commits that touched it under an
 * "Other changes" heading instead of asserting nothing happened.
 */
export default class TreeAwareChangelogRenderer extends DefaultChangelogRenderer {
  async render(): Promise<string> {
    // preprocessChanges() populates relevantChanges/breakingChanges from
    // this.changes; it's idempotent, so re-running it inside super.render()
    // when we delegate below does no harm.
    this.preprocessChanges();
    if (!this.shouldRenderEmptyEntry() || this.project === null) {
      return super.render();
    }

    const hiddenCommits = this.findHiddenCommitsTouchingProjectTree();
    if (hiddenCommits.length === 0) {
      return this.renderEmptyEntry();
    }

    return this.renderOtherChangesEntry(hiddenCommits);
  }

  /**
   * Returns the log lines of commits that touched the project's own path
   * between its previous release tag and HEAD, but only when that path
   * actually differs between the two -- i.e. only the cases the default
   * renderer gets wrong. An empty result means either there is no previous
   * tag to diff against (first release) or the tree genuinely didn't change,
   * both of which are cases where "no code changes" is true.
   */
  private findHiddenCommitsTouchingProjectTree(): string[] {
    const project = this.project as string;
    const projectRoot = this.findProjectRoot(project);
    if (!projectRoot) {
      return [];
    }

    const previousTag = this.findPreviousReleaseTag(project);
    // No previous tag reachable from before this release means this is the
    // project's first release: there is nothing prior to have hidden a
    // change from, so the default "no changes" behaviour is already correct.
    if (!previousTag) {
      return [];
    }

    const treeChanged = this.gitOutput([
      'diff',
      '--name-only',
      `${previousTag}..HEAD`,
      '--',
      projectRoot,
    ]);
    if (treeChanged === null || treeChanged.trim() === '') {
      return [];
    }

    const log = this.gitOutput([
      'log',
      '--oneline',
      `${previousTag}..HEAD`,
      '--',
      projectRoot,
    ]);
    if (log === null || log.trim() === '') {
      return [];
    }
    return log.trim().split('\n');
  }

  private findProjectRoot(project: string): string | null {
    try {
      const graph = readCachedProjectGraph();
      const node = graph?.nodes?.[project];
      return node?.data?.root ?? null;
    } catch {
      return null;
    }
  }

  /**
   * `git.commit: false` means the release never adds a commit of its own —
   * the new tag lands directly on HEAD. So "the previous release tag" is the
   * nearest ancestor of HEAD^ matching this project's tag pattern, found via
   * git's own reachability graph rather than a manual semver sort (which
   * would need an extra dependency neither this repo nor its lockfile
   * otherwise carries).
   */
  private findPreviousReleaseTag(project: string): string | null {
    const tag = this.gitOutput([
      'describe',
      '--tags',
      '--abbrev=0',
      '--match',
      `${project}-*`,
      'HEAD^',
    ]);
    const trimmed = tag?.trim();
    return trimmed ? trimmed : null;
  }

  private gitOutput(args: string[]): string | null {
    try {
      return execFileSync('git', args, { encoding: 'utf-8' });
    } catch {
      return null;
    }
  }

  private renderOtherChangesEntry(hiddenCommits: string[]): string {
    return [
      this.renderVersionTitle(),
      '',
      '### 🔧 Other changes',
      '',
      "This release carries no bump-triggering commits of its own, but the project's tree did change since its previous release. The changes below did not qualify for their own release and are listed here rather than reported as absent:",
      '',
      ...hiddenCommits.map((line) => `- ${line}`),
    ].join('\n');
  }
}
