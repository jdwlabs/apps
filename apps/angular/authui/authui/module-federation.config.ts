import { ModuleFederationConfig } from '@nx/webpack';

const config: ModuleFederationConfig = {
  name: 'authui',
  exposes: {
    './Routes':
      'apps/angular/authui/authui/src/app/remote-entry/entry.routes.ts',
  },
};

export default config;
