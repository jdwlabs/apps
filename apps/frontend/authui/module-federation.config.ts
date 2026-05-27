import { ModuleFederationConfig } from '@nx/module-federation';

const config: ModuleFederationConfig = {
  name: 'authui',
  exposes: {
    './Routes': 'apps/frontend/authui/src/app/remote-entry/entry.routes.ts',
  },
};

export default config;
