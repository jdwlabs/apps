import baseConfig from '../../../../eslint.config';
import nx from '@nx/eslint-plugin';
import jsoncParser from 'jsonc-eslint-parser';

export default [
  ...baseConfig,
  {
    files: ['**/*.json'],
    plugins: { '@nx': nx },
    languageOptions: { parser: jsoncParser },
    rules: { '@nx/dependency-checks': 'error' },
  },
];
