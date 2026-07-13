import nx from '@nx/eslint-plugin';
import globals from 'globals';

export default [
  { ignores: ['node_modules'] },
  {
    ignores: [
      '**/eslint.config.ts',
      '**/eslint.config.js',
      '**/eslint.config.mjs',
      // vite.config.ts is read directly by Vite (outside Nx's module graph,
      // before the tsconfig-paths plugin is active) and test-setup.ts is
      // Vitest tooling, not application code -- both import the shared
      // tools/testing helpers via relative path rather than a @jdw/* alias.
      '**/vite.config.ts',
      '**/src/test-setup.ts',
    ],
  },
  ...nx.configs['flat/base'],
  {
    files: ['**/*.ts', '**/*.tsx', '**/*.js', '**/*.jsx'],
    rules: {
      '@nx/enforce-module-boundaries': [
        'error',
        {
          enforceBuildableLibDependency: true,
          allow: [],
          // Three independent tag axes (type/scope/framework); a dependency
          // must satisfy every constraint that matches the importing project.
          // The full rules live in docs/conventions.md#module-boundaries.
          depConstraints: [
            // type axis: layering, high to low
            {
              sourceTag: 'type:app',
              onlyDependOnLibsWithTags: [
                'type:feature',
                'type:ui',
                'type:util',
                'type:data-access',
              ],
            },
            {
              sourceTag: 'type:feature',
              onlyDependOnLibsWithTags: [
                'type:ui',
                'type:util',
                'type:data-access',
              ],
            },
            {
              sourceTag: 'type:data-access',
              onlyDependOnLibsWithTags: ['type:util', 'type:data-access'],
            },
            {
              sourceTag: 'type:ui',
              onlyDependOnLibsWithTags: ['type:util'],
            },
            {
              sourceTag: 'type:util',
              onlyDependOnLibsWithTags: ['type:util'],
            },
            // E2E sits above everything and nothing depends on it, so it may
            // import from any project.
            {
              sourceTag: 'type:e2e',
              onlyDependOnLibsWithTags: ['*'],
            },
            // scope axis: each app scope sees only itself + shared
            {
              sourceTag: 'scope:shared',
              onlyDependOnLibsWithTags: ['scope:shared'],
            },
            {
              sourceTag: 'scope:authui',
              onlyDependOnLibsWithTags: ['scope:authui', 'scope:shared'],
            },
            {
              sourceTag: 'scope:container',
              onlyDependOnLibsWithTags: ['scope:container', 'scope:shared'],
            },
            {
              sourceTag: 'scope:rolesui',
              onlyDependOnLibsWithTags: ['scope:rolesui', 'scope:shared'],
            },
            {
              sourceTag: 'scope:usersui',
              onlyDependOnLibsWithTags: ['scope:usersui', 'scope:shared'],
            },
            {
              sourceTag: 'scope:servicediscovery',
              onlyDependOnLibsWithTags: [
                'scope:servicediscovery',
                'scope:shared',
              ],
            },
            {
              sourceTag: 'scope:usersrole',
              onlyDependOnLibsWithTags: ['scope:usersrole', 'scope:shared'],
            },
            {
              sourceTag: 'scope:authdb',
              onlyDependOnLibsWithTags: ['scope:authdb', 'scope:shared'],
            },
            {
              sourceTag: 'scope:ai-sre-relay',
              onlyDependOnLibsWithTags: ['scope:ai-sre-relay', 'scope:shared'],
            },
            // framework axis: no cross-framework imports. ESLint only lints
            // JS/TS, so this bites on the Angular/Node side; Go and JVM code
            // is additionally fenced by its own toolchain.
            {
              sourceTag: 'framework:angular',
              onlyDependOnLibsWithTags: ['framework:angular'],
            },
            {
              sourceTag: 'framework:go',
              onlyDependOnLibsWithTags: ['framework:go'],
            },
            {
              sourceTag: 'framework:springboot',
              onlyDependOnLibsWithTags: ['framework:springboot'],
            },
            {
              sourceTag: 'framework:database',
              onlyDependOnLibsWithTags: ['framework:database'],
            },
            {
              sourceTag: 'framework:node',
              onlyDependOnLibsWithTags: ['framework:node'],
            },
          ],
        },
      ],
    },
  },
  ...nx.configs['flat/typescript'],
  {
    files: ['**/*.ts', '**/*.tsx'],
    rules: { 'no-extra-semi': 'off' },
  },
  ...nx.configs['flat/javascript'],
  {
    files: ['**/*.js', '**/*.jsx'],
    rules: { 'no-extra-semi': 'off' },
  },
  {
    files: ['**/*.spec.ts', '**/*.spec.tsx', '**/*.spec.js', '**/*.spec.jsx'],
    languageOptions: { globals: { ...globals.vitest } },
  },
];
