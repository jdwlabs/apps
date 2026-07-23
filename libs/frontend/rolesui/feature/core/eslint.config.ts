import baseConfig from '../../../../../eslint.config';
import nx from '@nx/eslint-plugin';

export default [
  ...baseConfig,
  ...nx.configs['flat/angular'],
  ...nx.configs['flat/angular-template'],
  {
    files: ['**/*.ts'],
    rules: {
      '@angular-eslint/directive-selector': [
        'error',
        { type: 'attribute', prefix: 'lib', style: 'camelCase' },
      ],
      '@angular-eslint/component-selector': [
        'error',
        { type: 'element', prefix: 'lib', style: 'kebab-case' },
      ],
      // Angular v22's change-detection-eager migration sets an explicit
      // ChangeDetectionStrategy.Eager on every component to preserve pre-v22
      // behaviour, which this angular-eslint preset rule would otherwise flag.
      '@angular-eslint/prefer-on-push-component-change-detection': 'off',
    },
  },
];
