import type { ProjectGraph, Tree } from '@nx/devkit';
import { VersionActions } from 'nx/release';

/**
 * Versions projects that have no version manifest at all: the git tag is the
 * only record of a released version. Used by every app in the release set;
 * JS libs use Nx's built-in package.json actions instead.
 *
 * `validManifestFilenames = null` is what makes the manifest optional rather
 * than merely unused — Nx skips both manifest discovery and the existence
 * check when it is null, so nothing in the working tree has to change for a
 * release to happen, and therefore nothing has to be pushed to a protected
 * branch. Setting `manifestRootsToUpdate` for a project using these actions
 * would defeat that and fail during release-graph construction.
 */
export default class TagOnlyVersionActions extends VersionActions {
  validManifestFilenames = null;

  async readCurrentVersionFromSourceManifest(_tree: Tree): Promise<null> {
    return null;
  }

  async readCurrentVersionFromRegistry(
    _tree: Tree,
    _meta: object,
  ): Promise<null> {
    // No registry exists for these projects; the current version comes from
    // the git tag matching releaseTag.pattern.
    return null;
  }

  async readCurrentVersionOfDependency(): Promise<{
    currentVersion: string | null;
    dependencyCollection: string | null;
  }> {
    return { currentVersion: null, dependencyCollection: null };
  }

  async updateProjectVersion(
    _tree: Tree,
    _newVersion: string,
  ): Promise<string[]> {
    return [];
  }

  async updateProjectDependencies(
    _tree: Tree,
    _projectGraph: ProjectGraph,
    _deps: object,
  ): Promise<string[]> {
    // Nothing declares a dependency spec on these projects.
    return [];
  }
}
