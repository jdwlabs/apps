import { createTreeWithEmptyWorkspace } from '@nx/devkit/testing';
import TextFileVersionActions from './text-file-version-actions';

function makeActions(
  root: string,
  manifestPath: string,
): TextFileVersionActions {
  const actions = new TextFileVersionActions(
    { name: 'test-group' } as never,
    { name: 'demo', type: 'app', data: { root } } as never,
    {} as never,
  );
  actions.manifestsToUpdate = [
    { manifestPath, preserveLocalDependencyProtocols: false },
  ];
  return actions;
}

describe('TextFileVersionActions', () => {
  it('reads the trimmed current version from the manifest', async () => {
    const tree = createTreeWithEmptyWorkspace();
    tree.write('apps/demo/public/VERSION', '1.2.3\n');
    const actions = makeActions('apps/demo', 'apps/demo/public/VERSION');
    await expect(
      actions.readCurrentVersionFromSourceManifest(tree),
    ).resolves.toEqual({
      currentVersion: '1.2.3',
      manifestPath: 'apps/demo/public/VERSION',
    });
  });

  it('throws when the manifest is missing', async () => {
    const tree = createTreeWithEmptyWorkspace();
    const actions = makeActions('apps/demo', 'apps/demo/VERSION');
    await expect(
      actions.readCurrentVersionFromSourceManifest(tree),
    ).rejects.toThrow('apps/demo/VERSION');
  });

  it('writes the new version with trailing newline to every configured manifest', async () => {
    const tree = createTreeWithEmptyWorkspace();
    tree.write('apps/demo/VERSION', '1.2.3\n');
    const actions = makeActions('apps/demo', 'apps/demo/VERSION');
    const logs = await actions.updateProjectVersion(tree, '1.3.0');
    expect(tree.read('apps/demo/VERSION', 'utf-8')).toBe('1.3.0\n');
    expect(logs).toEqual(['Updated apps/demo/VERSION to 1.3.0']);
  });

  it('reports no registry and no dependency handling', async () => {
    const tree = createTreeWithEmptyWorkspace();
    const actions = makeActions('apps/demo', 'apps/demo/VERSION');
    await expect(
      actions.readCurrentVersionFromRegistry(tree, {}),
    ).resolves.toBeNull();
    await expect(
      actions.updateProjectDependencies(tree, {} as never, {}),
    ).resolves.toEqual([]);
  });
});
