import baseConfig from '../../../eslint.config';
import nx from '@nx/eslint-plugin';

export default [
  ...baseConfig,
  ...nx.configs['flat/typescript'],
  { files: ['**/*.ts'] },
];
