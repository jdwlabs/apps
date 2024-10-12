import { ModuleFederationConfig } from '@nx/webpack';

const config: ModuleFederationConfig = {
  name: 'usersui',
  exposes: {
    './Routes':
      'apps/angular/usersui/usersui/src/app/remote-entry/entry.routes.ts',
  },
};

export default config;
