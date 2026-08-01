import { createProjectGraphAsync } from '@nx/devkit';

/**
 * An Nx `inputs` array REPLACES the implicit `["default", "^default"]` rather
 * than extending it. A cacheable target that declares inputs without any
 * dependency-aware entry therefore hashes none of its dependencies: editing a
 * library cannot move the hash, the task hits cache, and the run reports a pass
 * for work it never did. That has shipped twice here — once via a hand-written
 * targetDefaults entry, once via a plugin's own defaults, where nothing in this
 * repo was edited at all.
 *
 * `^`-prefixed entries pull in dependency filesets. `dependentTasksOutputFiles`
 * hashes the artifacts dependencies produced, which is a stronger guarantee.
 */
const isDependencyAware = (input: unknown): boolean => {
  if (typeof input === 'string') {
    return input.startsWith('^');
  }
  if (input && typeof input === 'object') {
    return 'dependentTasksOutputFiles' in input || 'projects' in input;
  }
  return false;
};

/**
 * Targets exempted from the rule, each with the reason it cannot apply.
 * A target belongs here only when its output genuinely cannot depend on another
 * project — never merely because it is currently failing.
 */
const EXEMPT = new Map<string, string>();

describe('cacheable targets hash their dependencies', () => {
  it('every cacheable target with declared inputs is dependency-aware', async () => {
    const graph = await createProjectGraphAsync({ exitOnError: false });

    const offenders: string[] = [];

    for (const [projectName, node] of Object.entries(graph.nodes)) {
      for (const [targetName, target] of Object.entries(
        node.data.targets ?? {},
      )) {
        if (target.cache !== true) continue;

        // No declared inputs means the implicit ["default", "^default"] applies,
        // which is dependency-aware by construction.
        const inputs = target.inputs;
        if (!inputs || inputs.length === 0) continue;

        if (inputs.some(isDependencyAware)) continue;
        if (EXEMPT.has(targetName)) continue;

        offenders.push(
          `${projectName}:${targetName} — inputs: ${JSON.stringify(inputs)}`,
        );
      }
    }

    expect(
      offenders,
      'These cacheable targets declare inputs with no dependency-aware entry, so a ' +
        'change in a dependency cannot invalidate their cache. Add "^default" or ' +
        '"^production" to the inputs, or set cache: false if the target has side ' +
        'effects. If a target genuinely cannot depend on another project, add it to ' +
        'EXEMPT with the reason.\n' +
        offenders.join('\n'),
    ).toEqual([]);
  });
});
