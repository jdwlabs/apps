import { joinPathFragments, type ProjectGraph, type Tree } from '@nx/devkit';
import { VersionActions } from 'nx/release';

/**
 * Versions projects whose manifest is a plain-text VERSION file containing
 * only the semver string. Used by every app in the release set; JS libs use
 * Nx's built-in package.json actions instead. The manifest location comes
 * from release.version.manifestRootsToUpdate (defaults to the project root).
 */
export default class TextFileVersionActions extends VersionActions {
  validManifestFilenames = ['VERSION'];

  async readCurrentVersionFromSourceManifest(
    tree: Tree,
  ): Promise<{ currentVersion: string; manifestPath: string }> {
    const manifestPath =
      this.manifestsToUpdate[0]?.manifestPath ??
      joinPathFragments(this.projectGraphNode.data.root, 'VERSION');
    const contents = tree.read(manifestPath, 'utf-8');
    if (!contents || !contents.trim()) {
      throw new Error(`Unable to read a version from "${manifestPath}"`);
    }
    return { currentVersion: contents.trim(), manifestPath };
  }

  async readCurrentVersionFromRegistry(
    _tree: Tree,
    _meta: object,
  ): Promise<null> {
    // No registry exists for VERSION-file projects; current version comes
    // from git tags (or the manifest via the disk resolver).
    return null;
  }

  async readCurrentVersionOfDependency(): Promise<{
    currentVersion: string | null;
    dependencyCollection: string | null;
  }> {
    return { currentVersion: null, dependencyCollection: null };
  }

  async updateProjectVersion(
    tree: Tree,
    newVersion: string,
  ): Promise<string[]> {
    const logs: string[] = [];
    for (const { manifestPath } of this.manifestsToUpdate) {
      tree.write(manifestPath, `${newVersion}\n`);
      logs.push(`Updated ${manifestPath} to ${newVersion}`);
    }
    return logs;
  }

  async updateProjectDependencies(
    _tree: Tree,
    _projectGraph: ProjectGraph,
    _deps: object,
  ): Promise<string[]> {
    // VERSION files carry no dependency specs.
    return [];
  }
}
