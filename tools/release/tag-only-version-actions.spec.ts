import { createTreeWithEmptyWorkspace } from '@nx/devkit/testing';
import TagOnlyVersionActions from './tag-only-version-actions';

function makeActions(root: string): TagOnlyVersionActions {
  return new TagOnlyVersionActions(
    { name: 'test-group' } as never,
    { name: 'demo', type: 'app', data: { root } } as never,
    { manifestRootsToUpdate: [] } as never,
  );
}

describe('TagOnlyVersionActions', () => {
  it('declares no manifest filenames, which is what makes releases tree-free', () => {
    // Nx only skips manifest discovery and the existence check when this is
    // null. An empty array reads as "no filenames configured" and still walks
    // the manifest code path, so the distinction is load-bearing rather than
    // stylistic.
    expect(makeActions('apps/demo').validManifestFilenames).toBeNull();
  });

  it('resolves no manifest even when a VERSION file is present in the tree', async () => {
    // A leftover manifest must not become the version source again by accident:
    // the whole point is that the git tag wins.
    const tree = createTreeWithEmptyWorkspace();
    tree.write('apps/demo/VERSION', '1.2.3\n');
    await expect(
      makeActions('apps/demo').readCurrentVersionFromSourceManifest(tree),
    ).resolves.toBeNull();
  });

  it('leaves the tree untouched when a new version is applied', async () => {
    const tree = createTreeWithEmptyWorkspace();
    const before = tree.listChanges();
    const logs = await makeActions('apps/demo').updateProjectVersion(
      tree,
      '1.3.0',
    );
    expect(logs).toEqual([]);
    expect(tree.listChanges()).toEqual(before);
  });

  it('reports no registry and no dependency handling', async () => {
    const tree = createTreeWithEmptyWorkspace();
    const actions = makeActions('apps/demo');
    await expect(
      actions.readCurrentVersionFromRegistry(tree, {}),
    ).resolves.toBeNull();
    await expect(
      actions.updateProjectDependencies(tree, {} as never, {}),
    ).resolves.toEqual([]);
  });

  it('reports no dependency version, regardless of which dependency is asked about', async () => {
    // These projects carry no manifest-declared dependency specs, so there is
    // nothing for @nx/release's dependent-bump resolution to read here — it
    // must fall back to git-tag-based current-version resolution for any
    // dependent project.
    await expect(
      makeActions('apps/demo').readCurrentVersionOfDependency(),
    ).resolves.toEqual({
      currentVersion: null,
      dependencyCollection: null,
    });
  });
});
