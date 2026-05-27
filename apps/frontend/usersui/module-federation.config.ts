import { ModuleFederationConfig } from '@nx/module-federation';

const config: ModuleFederationConfig = {
  name: 'usersui',
  exposes: {
    './Routes': 'apps/frontend/usersui/src/app/remote-entry/entry.routes.ts',
  },
};

export default config;
