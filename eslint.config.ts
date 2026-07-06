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
          depConstraints: [
            {
              sourceTag: 'scope:shared',
              onlyDependOnLibsWithTags: ['scope:shared'],
            },
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
              onlyDependOnLibsWithTags: ['type:ui', 'type:util'],
            },
            {
              sourceTag: 'type:util',
              onlyDependOnLibsWithTags: ['type:util'],
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
