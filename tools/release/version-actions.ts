import type { Tree } from '@nx/devkit';
import JsVersionActions, {
  afterAllProjectsVersioned,
} from '@nx/js/src/release/version-actions';

// Re-exported so Nx Release keeps running the JS ecosystem's post-versioning
// steps (e.g. lock file maintenance) alongside our custom behaviour.
export { afterAllProjectsVersioned };

/**
 * Nx Release version actions for this workspace.
 *
 * The default JS implementation only writes `package.json`, but the deployed
 * apps read their version from other files. This extends the default bump to
 * also keep those files in lockstep:
 *
 * - Angular apps serve `public/VERSION` as a plain-text runtime asset.
 * - The Spring Boot app derives its jar and jib image version from the
 *   Gradle project version in `build.gradle.kts`.
 *
 * Both writes are conditional on the file existing, so the Go and database
 * apps (which only carry a version-tracking `package.json`) are unaffected.
 */
export default class WorkspaceVersionActions extends JsVersionActions {
  override async updateProjectVersion(
    tree: Tree,
    newVersion: string,
  ): Promise<string[]> {
    const logMessages = await super.updateProjectVersion(tree, newVersion);
    const projectRoot = this.projectGraphNode.data.root;

    const versionFile = `${projectRoot}/public/VERSION`;
    if (tree.exists(versionFile)) {
      tree.write(versionFile, `${newVersion}\n`);
      logMessages.push(
        `✍️  New version ${newVersion} written to ${versionFile}`,
      );
    }

    const gradleFile = `${projectRoot}/build.gradle.kts`;
    if (tree.exists(gradleFile)) {
      const contents = tree.read(gradleFile, 'utf-8');
      if (contents) {
        tree.write(
          gradleFile,
          contents.replace(/^version = ".*"$/m, `version = "${newVersion}"`),
        );
        logMessages.push(
          `✍️  New version ${newVersion} written to ${gradleFile}`,
        );
      }
    }

    return logMessages;
  }
}
